package awstooler

import (
	"context"
	"log/slog"
	"os/exec"
	"testing"

	"github.com/signoz/foundry/internal/tooler"
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

type otherTooler struct{}

func (otherTooler) Name() string                    { return "other" }
func (otherTooler) Gauge(ctx context.Context) error { return nil }

func TestNew(t *testing.T) {
	assert.Equal(t, "aws", New(slog.New(slog.DiscardHandler)).Name())
}

func TestLookup(t *testing.T) {
	aws := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		toolers []tooler.Tooler
		pass    bool
	}{
		{name: "Registered_Found", toolers: []tooler.Tooler{aws}, pass: true},
		{name: "AmongOthers_Found", toolers: []tooler.Tooler{otherTooler{}, aws}, pass: true},
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
			assert.Equal(t, aws, found)
		})
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
