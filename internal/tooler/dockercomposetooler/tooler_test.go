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
	"github.com/signoz/foundry/internal/tooler"
	"github.com/stretchr/testify/assert"
)

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

func TestUpDown(t *testing.T) {
	requireEngine(t)

	composeFile := filepath.Join(t.TempDir(), "compose.yaml")
	contents := "name: dockercomposetooler-test\nservices:\n  ok:\n    image: busybox:stable\n    command: [\"sleep\", \"300\"]\n"
	assert.NoError(t, os.WriteFile(composeFile, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	r.Settings = tooler.NewSettings(io.Discard)

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
	r.Settings = tooler.NewSettings(io.Discard)

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
