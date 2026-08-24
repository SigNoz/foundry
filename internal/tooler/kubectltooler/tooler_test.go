package kubectltooler

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
)

// requireKubectl skips a test that drives a real cluster through kubectl.
func requireKubectl(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping kubectl test in short mode")
	}

	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl is not available")
	}

	if err := exec.Command("kubectl", "cluster-info").Run(); err != nil {
		t.Skip("no reachable cluster")
	}
}

// kustomize writes a minimal kustomize root, so apply and delete touch only one
// ConfigMap.
func kustomize(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte("resources:\n- configmap.yaml\n"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "configmap.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: kubectltooler-test\ndata:\n  ok: \"yes\"\n"), 0o644))

	return dir
}

// Apply then Delete a one-ConfigMap kustomize; needs a reachable cluster, so it
// skips wherever one is absent.
func TestApplyDelete(t *testing.T) {
	requireKubectl(t)

	r := New(slog.New(slog.DiscardHandler))
	r.sink = io.Discard

	release := Release{
		Release: domain.Release{
			Name:  "kubectltooler-test",
			Owner: domain.Owner{"foundry.signoz.io/managed-by": "foundry"},
		},
		Dir: kustomize(t),
	}

	assert.NoError(t, r.Gauge(context.Background()))
	assert.NoError(t, r.Apply(context.Background(), release))
	assert.NoError(t, r.Delete(context.Background(), release))
}
