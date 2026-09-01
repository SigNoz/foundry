package helmtooler

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	helmrelease "helm.sh/helm/v3/pkg/release"
)

func TestValidate(t *testing.T) {
	complete := Release{
		Release:   domain.Release{Name: "signoz", Owner: domain.Owner{"foundry.signoz.io/name": "signoz"}},
		Namespace: "signoz",
		Chart:     "signoz/signoz",
	}

	without := func(mutate func(*Release)) Release {
		release := complete
		mutate(&release)

		return release
	}

	tests := []struct {
		name    string
		release Release
		pass    bool
	}{
		{name: "Complete_Valid", release: complete, pass: true},
		{name: "UnstatedName_Invalid", release: without(func(r *Release) { r.Name = "" })},
		{name: "UnstatedOwner_Invalid", release: without(func(r *Release) { r.Owner = nil })},
		{name: "UnstatedNamespace_Invalid", release: without(func(r *Release) { r.Namespace = "" })},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.release.Validate()

			if !tt.pass {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

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
