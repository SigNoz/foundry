package registry

import (
	"errors"

	"cuelang.org/go/cue"
	"github.com/signoz/foundry/internal/generator"
	"github.com/signoz/foundry/internal/generator/clickhouse"
	"github.com/signoz/foundry/internal/generator/signoz"
	"github.com/signoz/foundry/internal/generator/signozotelcollector"
	"github.com/signoz/foundry/internal/generator/zookeeper"
)

type ComponentID string

// Component identifiers.
const (
	ComponentClickHouse          ComponentID = "clickhouse"
	ComponentSignoz              ComponentID = "signoz"
	ComponentSignozOtelCollector ComponentID = "signozOtelCollector"
	ComponentZooKeeper           ComponentID = "zookeeper"
)

type ComponentRegistry struct {
	generators        map[ComponentID]generator.Generator
	config            cue.Value
	enabledComponents map[string]bool
}

func NewComponentRegistry(config cue.Value, enabledComponents map[string]bool) *ComponentRegistry {
	registry := &ComponentRegistry{
		generators:        make(map[ComponentID]generator.Generator),
		config:            config,
		enabledComponents: enabledComponents,
	}

	// register all components
	registry.registerAll()
	return registry
}

// registerAll registers all known components.
func (r *ComponentRegistry) registerAll() {
	r.register(ComponentClickHouse, &clickhouse.Generator{})
	r.register(ComponentSignoz, &signoz.Generator{})
	r.register(ComponentSignozOtelCollector, &signozotelcollector.Generator{})
	r.register(ComponentZooKeeper, &zookeeper.Generator{})
}

func (r *ComponentRegistry) register(id ComponentID, gen generator.Generator) {
	r.generators[id] = gen
}

// GetByID retrieves generator by ID.
func (r *ComponentRegistry) GetByID(id ComponentID) (generator.Generator, bool) {
	gen, ok := r.generators[id]
	return gen, ok
}


// GenerateFiles generates files for a component by ID.
func (r *ComponentRegistry) GenerateFiles(id ComponentID) (map[string][]byte, error) {
	gen, ok := r.GetByID(id)
	if !ok {
		return nil, errors.New("unknown component: " + string(id))
	}

	return gen.GenerateComponent(r.config)
}

func (r *ComponentRegistry) isComponentEnabled(id ComponentID) bool {
	if r.enabledComponents == nil {
		return false
	}
	// r.enabledComponents = "signoz": true
	return r.enabledComponents[string(id)]
}

// GenerateAllEnabled generates files for all enabled components.
func (r *ComponentRegistry) GenerateAllEnabled() (map[ComponentID]map[string][]byte, error) {
	results := make(map[ComponentID]map[string][]byte)

	for id := range r.generators {
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