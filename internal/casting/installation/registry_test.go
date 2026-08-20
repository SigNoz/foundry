package installation

import (
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/runner/composerunner"
	"github.com/stretchr/testify/assert"
)

// The compose casting resolves its runner from the registry entry at cast
// time; this pins the entry so the lookup cannot silently go empty.
func TestRegistryRunners(t *testing.T) {
	registry := NewRegistry(slog.New(slog.DiscardHandler))

	tests := []struct {
		name       string
		deployment v1alpha1.TypeDeployment
		pass       bool
	}{
		{name: "DockerCompose_ComposeRunner", deployment: v1alpha1.TypeDeployment{Mode: v1alpha1.ModeDocker, Flavor: v1alpha1.FlavorCompose}, pass: true},
		{name: "DockerSwarm_NoRunnerYet", deployment: v1alpha1.TypeDeployment{Mode: v1alpha1.ModeDocker, Flavor: v1alpha1.FlavorSwarm}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runners, err := registry.Runners(tt.deployment)
			assert.NoError(t, err)

			_, err = composerunner.Lookup(runners)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
