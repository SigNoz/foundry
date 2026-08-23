package helmtooler

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/stretchr/testify/assert"
	helmrelease "helm.sh/helm/v3/pkg/release"
)

type otherTooler struct{}

func (otherTooler) Name() string                    { return "other" }
func (otherTooler) Gauge(ctx context.Context) error { return nil }

func TestNew(t *testing.T) {
	r := New(slog.New(slog.DiscardHandler))

	assert.Equal(t, "helm", r.Name())
}

// Gauge has no reach check: helm is the SDK, so there is no binary to find and
// a preflight always passes.
func TestGaugeNeedsNoBinary(t *testing.T) {
	r := New(slog.New(slog.DiscardHandler))

	assert.NoError(t, r.Gauge(context.Background()))
}

func TestLookup(t *testing.T) {
	helm := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		toolers []tooler.Tooler
		pass    bool
	}{
		{name: "Registered_Found", toolers: []tooler.Tooler{helm}, pass: true},
		{name: "AmongOthers_Found", toolers: []tooler.Tooler{otherTooler{}, helm}, pass: true},
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
			assert.Equal(t, helm, found)
		})
	}
}

// A verb that names neither release nor namespace refuses before it reaches
// the cluster, the same statement check every mutating verb owns.
func TestVerbsRequireReleaseAndNamespace(t *testing.T) {
	helm := New(slog.New(slog.DiscardHandler))

	assert.Error(t, helm.Upgrade(context.Background(), Release{Chart: "signoz/signoz"}))
	assert.Error(t, helm.Uninstall(context.Background(), Release{}))
}

// Ownership compares only the attributes foundry stamps: a release also
// carries helm's own system labels, and a foreign owner blocks the call the
// way the compose project guard does.
func TestOwnerGuard(t *testing.T) {
	helm := New(slog.New(slog.DiscardHandler))

	owner := domain.Owner{
		"foundry.signoz.io/managed-by": "foundry",
		"foundry.signoz.io/name":       "signoz",
	}

	tests := []struct {
		name   string
		labels map[string]string
		pass   bool
	}{
		{
			name:   "SameOwner_Allowed",
			labels: map[string]string{"foundry.signoz.io/managed-by": "foundry", "foundry.signoz.io/name": "signoz", "owner": "helm", "status": "deployed"},
			pass:   true,
		},
		{
			name:   "ForeignOwner_Refused",
			labels: map[string]string{"foundry.signoz.io/managed-by": "foundry", "foundry.signoz.io/name": "other"},
			pass:   false,
		},
		{
			name:   "Unowned_Allowed",
			labels: map[string]string{"owner": "helm", "status": "deployed"},
			pass:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := helm.verify(context.Background(), &helmrelease.Release{Name: "signoz", Labels: tt.labels}, owner)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
