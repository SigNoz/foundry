package composerunner

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

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/runner"
	"github.com/stretchr/testify/assert"
)

// requireEngine skips a test that drives a real docker engine.
func requireEngine(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping docker engine test in short mode")
	}

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not available")
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker engine is not running")
	}
}

type otherRunner struct{}

func (otherRunner) Name() string                    { return "other" }
func (otherRunner) Gauge(ctx context.Context) error { return nil }

func TestNew(t *testing.T) {
	r := New(slog.New(slog.DiscardHandler))

	assert.Equal(t, "docker compose", r.Name())
}

func TestLookup(t *testing.T) {
	compose := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		runners []runner.Runner
		pass    bool
	}{
		{name: "Registered_Found", runners: []runner.Runner{compose}, pass: true},
		{name: "AmongOthers_Found", runners: []runner.Runner{otherRunner{}, compose}, pass: true},
		{name: "Empty_Invalid", runners: nil, pass: false},
		{name: "OnlyOthers_Invalid", runners: []runner.Runner{otherRunner{}}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := Lookup(tt.runners)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, compose, found)
		})
	}
}

// The verbs validate what they execute against, so a casting that forgot to
// state the file, or forged nothing, fails before the tool is spawned.
func TestRunValidatesFile(t *testing.T) {
	tests := []struct {
		name string
		file string
		pass bool
	}{
		{name: "Unset_Invalid", file: "", pass: false},
		{name: "Missing_Invalid", file: filepath.Join(t.TempDir(), "absent", "compose.yaml"), pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(slog.New(slog.DiscardHandler))

			if !tt.pass {
				assert.Error(t, r.Up(context.Background(), Options{File: tt.file}))
				assert.Error(t, r.Down(context.Background(), Options{File: tt.file}))
				return
			}

			assert.NoError(t, r.Up(context.Background(), Options{File: tt.file}))
		})
	}
}

// owners reads docker's flat output back into the owners domain compares.
// The encoding is the runner's: a container reporting nothing must read as
// unowned, not as an owner whose every value is empty.
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
			out:  "Installation|foundry|signoz\n",
			expectedOwners: []domain.Owner{{
				"foundry.signoz.io/kind":       "Installation",
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
			assert.Equal(t, tt.expectedOwners, owners(keys, tt.out))
		})
	}

	// A container reporting nothing marks the group unowned rather than
	// conflicting with the caller.
	ownership := domain.NewOwnership(owners(keys, "||\n")...)
	_, conflict := ownership.Foreign(domain.Owner{"foundry.signoz.io/kind": "Installation"})

	assert.False(t, conflict)
	assert.True(t, ownership.HasUnowned())
}

// Up then Down against a minimal compose file; needs a running docker engine,
// so it skips wherever one is absent.
func TestUpDown(t *testing.T) {
	requireEngine(t)

	composeFile := filepath.Join(t.TempDir(), "compose.yaml")
	contents := "name: composerunner-test\nservices:\n  ok:\n    image: busybox:stable\n    command: [\"sleep\", \"300\"]\n"
	assert.NoError(t, os.WriteFile(composeFile, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))

	assert.NoError(t, r.Gauge(context.Background()))
	assert.NoError(t, r.Up(context.Background(), Options{File: composeFile, Stdout: io.Discard, Stderr: io.Discard}))
	assert.NoError(t, r.Down(context.Background(), Options{File: composeFile, Stdout: io.Discard, Stderr: io.Discard}))
}

// A project labelled for one owner is refused to another, and granted back to
// the owner that holds it. Needs a running docker engine.
func TestOwnerGuardsTheProject(t *testing.T) {
	requireEngine(t)

	const project = "composerunner-owner-test"

	owner := domain.Owner{
		"foundry.signoz.io/managed-by": "foundry",
		"foundry.signoz.io/kind":       "Installation",
		"foundry.signoz.io/name":       project,
	}

	labels := strings.Builder{}
	for key, value := range owner {
		labels.WriteString("      " + key + ": " + value + "\n")
	}

	composeFile := filepath.Join(t.TempDir(), "compose.yaml")
	contents := "name: " + project + "\nservices:\n  ok:\n    image: busybox:stable\n    command: [\"sleep\", \"300\"]\n    labels:\n" + labels.String()
	assert.NoError(t, os.WriteFile(composeFile, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	options := Options{
		File:    composeFile,
		Project: project,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	}

	installation := options
	installation.Owner = owner

	// One label of the set differing is a different owner.
	agent := options
	agent.Owner = maps.Clone(owner)
	agent.Owner["foundry.signoz.io/kind"] = "CollectionAgent"

	assert.NoError(t, r.Up(context.Background(), installation))
	t.Cleanup(func() { _ = r.Down(context.Background(), options) })

	assert.Error(t, r.Up(context.Background(), agent))
	assert.Error(t, r.Down(context.Background(), agent))
	assert.NoError(t, r.Down(context.Background(), installation))
}
