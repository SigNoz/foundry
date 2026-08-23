package dockerswarmtooler

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/stretchr/testify/assert"
)

// requireSwarm skips a test that drives a real docker swarm manager.
func requireSwarm(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping docker swarm test in short mode")
	}

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not available")
	}

	out, err := exec.Command("docker", "info", "--format", "{{.Swarm.ControlAvailable}}").Output()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		t.Skip("docker swarm manager is not available")
	}
}

// waitForStack polls until the stack's task container is scheduled, since stack
// deploy returns before its tasks converge.
func waitForStack(t *testing.T, stack string) {
	t.Helper()

	for range 60 {
		out, _ := exec.Command("docker", "ps", "-a", "--filter", "label=com.docker.stack.namespace="+stack, "--format", "{{.ID}}").Output()
		if strings.TrimSpace(string(out)) != "" {
			return
		}

		time.Sleep(time.Second)
	}

	t.Fatal("stack container did not appear")
}

type otherTooler struct{}

func (otherTooler) Name() string                    { return "other" }
func (otherTooler) Gauge(ctx context.Context) error { return nil }

func TestNew(t *testing.T) {
	r := New(slog.New(slog.DiscardHandler))

	assert.Equal(t, "docker stack", r.Name())
}

func TestLookup(t *testing.T) {
	swarm := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		toolers []tooler.Tooler
		pass    bool
	}{
		{name: "Registered_Found", toolers: []tooler.Tooler{swarm}, pass: true},
		{name: "AmongOthers_Found", toolers: []tooler.Tooler{otherTooler{}, swarm}, pass: true},
		{name: "Empty_Invalid", toolers: nil, pass: false},
		{name: "OnlyOthers_Invalid", toolers: []tooler.Tooler{otherTooler{}}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := Lookup(tt.toolers)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, swarm, found)
		})
	}
}

// The mutating verbs state what they act on before the tool is spawned: an
// unstated file has no stack to deploy, and an unstated claim would silently
// skip the owner guard.
func TestMutateStatesTheRelease(t *testing.T) {
	claim := domain.Owner{"foundry.signoz.io/managed-by": "foundry"}

	tests := []struct {
		name    string
		release Release
	}{
		{name: "UnstatedFile_Invalid", release: Release{Release: domain.Release{Name: "signoz", Owner: claim}}},
		{name: "UnstatedClaim_Invalid", release: Release{File: "compose.yaml"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(slog.New(slog.DiscardHandler))

			assert.Error(t, r.Up(context.Background(), tt.release))
			assert.Error(t, r.Down(context.Background(), tt.release))
		})
	}
}

// parse reads docker's flat output back into the owners domain compares. A
// container reporting nothing must read as unowned, not as an owner whose every
// value is empty.
func TestOwners(t *testing.T) {
	keys := []string{"foundry.signoz.io/kind", "foundry.signoz.io/managed-by", "foundry.signoz.io/name"}

	tests := []struct {
		name           string
		out            string
		expectedOwners []domain.Owner
	}{
		{name: "NoContainers_None", out: "", expectedOwners: []domain.Owner{}},
		{
			name: "OneContainer_OneOwner",
			out:  "CollectionAgent|foundry|signoz\n",
			expectedOwners: []domain.Owner{{
				"foundry.signoz.io/kind":       "CollectionAgent",
				"foundry.signoz.io/managed-by": "foundry",
				"foundry.signoz.io/name":       "signoz",
			}},
		},
		{
			name: "NoLabels_ZeroOwner",
			out:  "||\n",
			expectedOwners: []domain.Owner{{
				"foundry.signoz.io/kind":       "",
				"foundry.signoz.io/managed-by": "",
				"foundry.signoz.io/name":       "",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedOwners, parse(keys, tt.out))
		})
	}

	ownership := domain.NewOwnership(parse(keys, "||\n")...)
	_, conflict := ownership.Foreign(domain.Owner{"foundry.signoz.io/kind": "Installation"})

	assert.False(t, conflict)
	assert.True(t, ownership.HasUnowned())
}

// Up then Down against a minimal stack; needs a running swarm manager, so it
// skips wherever one is absent.
func TestUpDown(t *testing.T) {
	requireSwarm(t)

	composeFile := filepath.Join(t.TempDir(), "compose.yaml")
	contents := "version: \"3\"\nservices:\n  ok:\n    image: busybox:stable\n    command: [\"sleep\", \"300\"]\n"
	assert.NoError(t, os.WriteFile(composeFile, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	r.sink = io.Discard

	release := Release{
		Release: domain.Release{
			Name: "dockerswarmtooler-test",
			Owner: domain.Owner{
				"foundry.signoz.io/managed-by": "foundry",
				"foundry.signoz.io/kind":       "Installation",
				"foundry.signoz.io/name":       "dockerswarmtooler-test",
			},
		},
		File: composeFile,
	}

	assert.NoError(t, r.Gauge(context.Background()))
	assert.NoError(t, r.Up(context.Background(), release))
	t.Cleanup(func() { _ = r.Down(context.Background(), release) })
	assert.NoError(t, r.Down(context.Background(), release))
}

// A stack labelled for one owner is refused to another, and granted back to the
// owner that holds it. Needs a running swarm manager.
func TestOwnerGuardsTheStack(t *testing.T) {
	requireSwarm(t)

	const stack = "dockerswarmtooler-owner-test"

	owner := domain.Owner{
		"foundry.signoz.io/managed-by": "foundry",
		"foundry.signoz.io/kind":       "Installation",
		"foundry.signoz.io/name":       stack,
	}

	labels := strings.Builder{}
	for key, value := range owner {
		labels.WriteString("      " + key + ": " + value + "\n")
	}

	composeFile := filepath.Join(t.TempDir(), "compose.yaml")
	contents := "version: \"3\"\nservices:\n  ok:\n    image: busybox:stable\n    command: [\"sleep\", \"300\"]\n    labels:\n" + labels.String()
	assert.NoError(t, os.WriteFile(composeFile, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	r.sink = io.Discard

	release := Release{
		Release: domain.Release{Name: stack},
		File:    composeFile,
	}

	installation := release
	installation.Owner = owner

	// One label of the set differing is a different owner.
	agent := release
	agent.Owner = maps.Clone(owner)
	agent.Owner["foundry.signoz.io/kind"] = "CollectionAgent"

	assert.NoError(t, r.Up(context.Background(), installation))
	t.Cleanup(func() { _ = r.Down(context.Background(), installation) })
	waitForStack(t, stack)

	assert.Error(t, r.Up(context.Background(), agent))
	assert.Error(t, r.Down(context.Background(), agent))
	assert.NoError(t, r.Down(context.Background(), installation))
}
