package awstooler

import (
	"context"
	"log/slog"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// requireAWS skips a test that drives the real aws binary.
func requireAWS(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping aws test in short mode")
	}

	if _, err := exec.LookPath("aws"); err != nil {
		t.Skip("aws is not available")
	}
}

// UpdateKubeconfig names the cluster whose context it writes, so without one it
// refuses before it reaches the CLI.
func TestUpdateKubeconfigRequiresCluster(t *testing.T) {
	aws := New(slog.New(slog.DiscardHandler))

	assert.Error(t, aws.UpdateKubeconfig(context.Background(), Options{}))
}

// Gauge resolves aws and proves it converses; it needs the real binary, so it
// skips wherever one is absent.
func TestGaugeReachesAWS(t *testing.T) {
	requireAWS(t)

	assert.NoError(t, New(slog.New(slog.DiscardHandler)).Gauge(context.Background()))
}
