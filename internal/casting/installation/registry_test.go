package installation

import (
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/tooler/dockercomposetooler"
	"github.com/stretchr/testify/assert"
)

// The compose casting resolves its tooler from the registry entry at cast
// time; this pins the entry so the lookup cannot silently go empty.
func TestRegistryToolers(t *testing.T) {
	registry := NewRegistry(slog.New(slog.DiscardHandler))

	tests := []struct {
		name       string
		deployment v1alpha1.TypeDeployment
		pass       bool
	}{
		{name: "DockerCompose_Valid", deployment: v1alpha1.TypeDeployment{Mode: v1alpha1.ModeDocker, Flavor: v1alpha1.FlavorCompose}, pass: true},
		{name: "DockerSwarm_Invalid", deployment: v1alpha1.TypeDeployment{Mode: v1alpha1.ModeDocker, Flavor: v1alpha1.FlavorSwarm}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolers, err := registry.Toolers(tt.deployment)
			assert.NoError(t, err)

			_, err = dockercomposetooler.Lookup(toolers)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
