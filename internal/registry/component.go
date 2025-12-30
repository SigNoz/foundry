package registry

import (
	"errors"
	"github.com/SigNoz/foundry/internal/generator"
	"github.com/SigNoz/foundry/internal/generator/clickhouse"
	"github.com/SigNoz/foundry/internal/generator/signoz"
	"github.com/SigNoz/foundry/internal/generator/signozotelcollector"
	"github.com/SigNoz/foundry/internal/generator/zookeeper"
	"github.com/SigNoz/foundry/internal/schema"
)

// ComponentID represents a type-safe component identifier
type ComponentID string

// Component identifiers from CUE schema
const (
	ComponentClickHouse          ComponentID = "clickhouse"
	ComponentSignoz              ComponentID = "signoz"
	ComponentSignozOtelCollector ComponentID = "signozOtelCollector"
	ComponentZooKeeper           ComponentID = "zookeeper"
)

// ComponentDefinition links component with its generator
type ComponentDefinition struct {
	ID       ComponentID
	Generator generator.Generator
}

// ComponentRegistry manages component generators with typed config
type ComponentRegistry struct {
	components    map[ComponentID]*ComponentDefinition
	castingConfig casting.Config
	enabledComponents map[string]bool
}

// NewComponentRegistry creates a new component registry with typed config
func NewComponentRegistry(config casting.Config, enabledComponents map[string]bool) *ComponentRegistry {
	registry := &ComponentRegistry{
		components:       make(map[ComponentID]*ComponentDefinition),
		castingConfig:    config,
		enabledComponents: enabledComponents,
	}
	
	// Auto-register all components
	registry.registerAll()
	return registry
}

// registerAll registers all known components
func (r *ComponentRegistry) registerAll() {
	r.register(&ComponentDefinition{
		ID:       ComponentClickHouse,
		Generator: &clickhouse.Generator{},
	})
	
	r.register(&ComponentDefinition{
		ID:       ComponentSignoz,
		Generator: &signoz.Generator{},
	})
	
	r.register(&ComponentDefinition{
		ID:       ComponentSignozOtelCollector,
		Generator: &signozotelcollector.Generator{},
	})
	
	r.register(&ComponentDefinition{
		ID:       ComponentZooKeeper,
		Generator: &zookeeper.Generator{},
	})
}

// register adds a component to the registry
func (r *ComponentRegistry) register(def *ComponentDefinition) {
	r.components[def.ID] = def
}

// GetByID retrieves component by type-safe ID
func (r *ComponentRegistry) GetByID(id ComponentID) (*ComponentDefinition, bool) {
	def, ok := r.components[id]
	return def, ok
}


// GenerateFiles generates files for a component by name
func (r *ComponentRegistry) GenerateFiles(id ComponentID) (map[string][]byte, error) {
	def, ok := r.GetByID(id)
	if !ok {
		return nil, errors.New("unknown component: " + string(id))
	}
	
	return def.Generator.GenerateComponent(r.castingConfig)
}

// isComponentEnabled checks if a component is enabled in the configuration
func (r *ComponentRegistry) isComponentEnabled(id ComponentID) bool {
	if r.enabledComponents == nil {
		return false
	}
	// r.enabledComponents = "signoz": true
	return r.enabledComponents[string(id)]
}

// GenerateAllEnabled generates files for all enabled components
func (r *ComponentRegistry) GenerateAllEnabled() (map[ComponentID]map[string][]byte, error) {
	results := make(map[ComponentID]map[string][]byte)
	
	for id := range r.components {
		// Check if component is enabled in config
		if !r.isComponentEnabled(id) {
			continue
		}
		
		files, err := r.GenerateFiles(id)
		if err != nil {
			return nil, errors.New("failed to generate " + string(id) + ": " + err.Error())
		}
		results[id] = files
	}
	
	return results, nil
}