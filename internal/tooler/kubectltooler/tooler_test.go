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
	"github.com/signoz/foundry/internal/tooler"
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

type otherTooler struct{}

func (otherTooler) Name() string                    { return "other" }
func (otherTooler) Gauge(ctx context.Context) error { return nil }

func TestNew(t *testing.T) {
	assert.Equal(t, "kubectl", New(slog.New(slog.DiscardHandler)).Name())
}

func TestLookup(t *testing.T) {
	kubectl := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		toolers []tooler.Tooler
		pass    bool
	}{
		{name: "Registered_Found", toolers: []tooler.Tooler{kubectl}, pass: true},
		{name: "AmongOthers_Found", toolers: []tooler.Tooler{otherTooler{}, kubectl}, pass: true},
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
			assert.Equal(t, kubectl, found)
		})
	}
}

// The mutating verbs state what they act on before kubectl is spawned: an
// unstated directory has nothing to apply, and an unstated claim leaves the
// deployable unit unidentified.
func TestMutateStatesTheRelease(t *testing.T) {
	claim := domain.Owner{"foundry.signoz.io/managed-by": "foundry"}

	tests := []struct {
		name    string
		release Release
	}{
		{name: "UnstatedDir_Invalid", release: Release{Release: domain.Release{Name: "signoz", Owner: claim}}},
		{name: "UnstatedClaim_Invalid", release: Release{Dir: "/tmp/kustomize"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(slog.New(slog.DiscardHandler))

			assert.Error(t, r.Apply(context.Background(), tt.release))
			assert.Error(t, r.Delete(context.Background(), tt.release))
		})
	}
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
