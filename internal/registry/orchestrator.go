package registry

import (
	"fmt"

	"cuelang.org/go/cue"
	"github.com/signoz/foundry/internal/orchestrator"
	casting "github.com/signoz/foundry/internal/schema"
)

// PlatformID represents a type-safe platform identifier
type PlatformID string

// Platform identifiers from CUE schema
const (
	PlatformDocker PlatformID = "docker"
	PlatformLinux  PlatformID = "linux"
	// Add more platforms as needed
)

// OrchestratorDefinition links platform with its orchestrator
type OrchestratorDefinition struct {
	ID          PlatformID
	Name        string
	Orchestrator orchestrator.Orchestrator
}

// OrchestratorRegistry manages platform-specific orchestrators
type OrchestratorRegistry struct {
	orchestrators map[PlatformID]*OrchestratorDefinition
	castingConfig casting.Config
}

// NewOrchestratorRegistry creates a new orchestrator registry with unified config
func NewOrchestratorRegistry(config casting.Config) *OrchestratorRegistry {
	registry := &OrchestratorRegistry{
		orchestrators: make(map[PlatformID]*OrchestratorDefinition),
		castingConfig: config,
	}
	
	// Auto-register all orchestrators
	registry.registerAll()
	return registry
}

// registerAll registers all known orchestrators
func (r *OrchestratorRegistry) registerAll() {
	// r.register(&OrchestratorDefinition{
	// 	ID:          PlatformDocker,
	// 	Name:        "Docker Compose",
	// 	Orchestrator: &orchestrator.DockerOrchestrator{},
	// })
	
	// r.register(&OrchestratorDefinition{
	// 	ID:          PlatformLinux,
	// 	Name:        "Systemd",
	// 	Orchestrator: &orchestrator.SystemdOrchestrator{},
	// })
}

// register adds an orchestrator to the registry
func (r *OrchestratorRegistry) register(def *OrchestratorDefinition) {
	r.orchestrators[def.ID] = def
}

// GetByID retrieves orchestrator by type-safe ID
func (r *OrchestratorRegistry) GetByID(id PlatformID) (*OrchestratorDefinition, bool) {
	def, ok := r.orchestrators[id]
	return def, ok
}

// GetByName retrieves orchestrator by string name (for config compatibility)
func (r *OrchestratorRegistry) GetByName(name string) (*OrchestratorDefinition, bool) {
	for _, def := range r.orchestrators {
		if string(def.ID) == name {
			return def, true
		}
	}
	return nil, false
}

// GenerateFiles generates orchestration files for a platform
func (r *OrchestratorRegistry) GenerateFiles(platform string, config cue.Value, enabledComponents []string) (map[string][]byte, error) {
	def, ok := r.GetByName(platform)
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	
	return def.Orchestrator.Generate(config, enabledComponents)
}
