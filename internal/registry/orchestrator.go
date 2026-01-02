package registry

import (
	"fmt"

	"cuelang.org/go/cue"
	"github.com/signoz/foundry/internal/orchestrator"
	"github.com/signoz/foundry/internal/orchestrator/dockercompose"
	"github.com/signoz/foundry/internal/orchestrator/systemd"
)

type PlatformID string

// Platform identifiers from CUE schema.
const (
	PlatformDocker PlatformID = "docker"
	PlatformLinux  PlatformID = "linux"
)

// OrchestratorRegistry manages platform-specific orchestrators.
type OrchestratorRegistry struct {
	orchestrators map[PlatformID]orchestrator.Orchestrator
	castingConfig cue.Value
}

func NewOrchestratorRegistry(config cue.Value) *OrchestratorRegistry {
	registry := &OrchestratorRegistry{
		orchestrators: make(map[PlatformID]orchestrator.Orchestrator),
		castingConfig: config,
	}

	registry.registerAll()
	return registry
}

// registerAll registers all known orchestrators.
func (r *OrchestratorRegistry) registerAll() {
	r.register(PlatformDocker, &dockercompose.Orchestrator{})
	r.register(PlatformLinux, &systemd.Orchestrator{})
}

// register adds an orchestrator to the registry.
func (r *OrchestratorRegistry) register(id PlatformID, orch orchestrator.Orchestrator) {
	r.orchestrators[id] = orch
}

// GetByID retrieves orchestrator by type-safe ID.
func (r *OrchestratorRegistry) GetByID(id PlatformID) (orchestrator.Orchestrator, bool) {
	orch, ok := r.orchestrators[id]
	return orch, ok
}

func (r *OrchestratorRegistry) GetByName(name string) (orchestrator.Orchestrator, bool) {
	orch, ok := r.orchestrators[PlatformID(name)]
	return orch, ok
}

func (r *OrchestratorRegistry) Generate(platform string, enabledComponents []string) (map[string][]byte, error) {
	orch, ok := r.GetByName(platform)
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return orch.Generate(r.castingConfig, enabledComponents)
}
