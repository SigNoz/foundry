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

func TestUpDown(t *testing.T) {
	requireSwarm(t)

	composeFile := filepath.Join(t.TempDir(), "compose.yaml")
	contents := "version: \"3\"\nservices:\n  ok:\n    image: busybox:stable\n    command: [\"sleep\", \"300\"]\n"
	assert.NoError(t, os.WriteFile(composeFile, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	r.Settings = tooler.NewSettings(io.Discard)

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
	r.Settings = tooler.NewSettings(io.Discard)

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

	assert.ErrorContains(t, r.Up(context.Background(), agent), "already belongs to")
	assert.ErrorContains(t, r.Down(context.Background(), agent), "already belongs to")
	assert.NoError(t, r.Down(context.Background(), installation))
}
