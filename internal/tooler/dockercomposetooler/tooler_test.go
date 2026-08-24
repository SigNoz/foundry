package dockercomposetooler

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

// parse reads docker's flat output back into the owners domain compares.
// The encoding is the tooler's: a container reporting nothing must read as
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
			assert.Equal(t, tt.expectedOwners, parse(keys, tt.out))
		})
	}

	// A container reporting nothing marks the group unowned rather than
	// conflicting with the caller.
	ownership := domain.NewOwnership(parse(keys, "||\n")...)
	_, conflict := ownership.Foreign(domain.Owner{"foundry.signoz.io/kind": "Installation"})

	assert.False(t, conflict)
	assert.True(t, ownership.HasUnowned())
}

// Up then Down against a minimal compose file; needs a running docker engine,
// so it skips wherever one is absent.
func TestUpDown(t *testing.T) {
	requireEngine(t)

	composeFile := filepath.Join(t.TempDir(), "compose.yaml")
	contents := "name: dockercomposetooler-test\nservices:\n  ok:\n    image: busybox:stable\n    command: [\"sleep\", \"300\"]\n"
	assert.NoError(t, os.WriteFile(composeFile, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	r.sink = io.Discard

	release := Release{
		Release: domain.Release{
			Name: "dockercomposetooler-test",
			Owner: domain.Owner{
				"foundry.signoz.io/managed-by": "foundry",
				"foundry.signoz.io/kind":       "Installation",
				"foundry.signoz.io/name":       "dockercomposetooler-test",
			},
		},
		File: composeFile,
	}

	assert.NoError(t, r.Gauge(context.Background()))
	assert.NoError(t, r.Up(context.Background(), release))
	assert.NoError(t, r.Down(context.Background(), release))
}

// A project labelled for one owner is refused to another, and granted back to
// the owner that holds it. Needs a running docker engine.
func TestOwnerGuardsTheProject(t *testing.T) {
	requireEngine(t)

	const project = "dockercomposetooler-owner-test"

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
	r.sink = io.Discard

	release := Release{
		Release: domain.Release{Name: project},
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

	assert.Error(t, r.Up(context.Background(), agent))
	assert.Error(t, r.Down(context.Background(), agent))
	assert.NoError(t, r.Down(context.Background(), installation))
}
