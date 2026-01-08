package registry

import (
	"errors"
	"cuelang.org/go/cue"
	"github.com/signoz/foundry/internal/generator"
	"github.com/signoz/foundry/internal/generator/clickhouse"
	"github.com/signoz/foundry/internal/generator/postgres"
	"github.com/signoz/foundry/internal/generator/signoz"
	"github.com/signoz/foundry/internal/generator/signozotelcollector"
	"github.com/signoz/foundry/internal/generator/linux"
)

// SigNoz Component identifiers.
const (
	ComponentClickHouse          generator.ComponentID = "clickhouse"
	ComponentSignoz              generator.ComponentID = "signoz"
	ComponentSignozOtelCollector generator.ComponentID = "signozOtelCollector"
	ComponentPostgres            generator.ComponentID = "postgres"
)

// Platform identifiers.
const (
	PlatformLinux generator.PlatformID = "linux"
)

// ComponentGeneratorRegistry manages all component generators.
type ComponentRegistry struct {
	generators map[generator.ComponentID]generator.ComponentGenerator
}

type PlatformRegistry struct {
	generators map[generator.PlatformID]generator.PlatformGenerator
}

// NewComponentGeneratorRegistry creates a new component generator registry with all known components.
func NewComponentRegistry() *ComponentRegistry {
	registry := &ComponentRegistry{
		generators: make(map[generator.ComponentID]generator.ComponentGenerator),
	}

	// Register all component generators
	registry.registerAll()
	return registry
}

// registerAll registers all known component generators.
func (r *ComponentRegistry) registerAll() {
	r.register(ComponentClickHouse, &clickhouse.Generator{})
	r.register(ComponentSignoz, &signoz.Generator{})
	r.register(ComponentSignozOtelCollector, &signozotelcollector.Generator{})
	r.register(ComponentPostgres, &postgres.Generator{})
}

func (r *ComponentRegistry) register(id generator.ComponentID, gen generator.ComponentGenerator) {
	r.generators[id] = gen
}

// Get retrieves a component generator by ComponentID.
func (r *ComponentRegistry) Get(id generator.ComponentID) (generator.ComponentGenerator, bool) {
	gen, ok := r.generators[id]
	return gen, ok
}

// GetAll returns all registered component generators keyed by ComponentID.
func (r *ComponentRegistry) GetAll() map[generator.ComponentID]generator.ComponentGenerator {
	return r.generators
}

// NewPlatformGeneratorRegistry creates a new platform generator registry with all known platforms.
func NewPlatformRegistry() *PlatformRegistry {
	registry := &PlatformRegistry{
		generators: make(map[generator.PlatformID]generator.PlatformGenerator),
	}

	// Register all platform generators
	registry.registerAll()
	return registry
}

// registerAll registers all known platform generators.
func (r *PlatformRegistry) registerAll() {
	r.register(PlatformLinux, &linux.PlatformGenerator{})
}

func (r *PlatformRegistry) register(id generator.PlatformID, gen generator.PlatformGenerator) {
	r.generators[id] = gen
}

// Get retrieves a platform generator by PlatformID.
func (r *PlatformRegistry) Get(id generator.PlatformID) (generator.PlatformGenerator, bool) {
	gen, ok := r.generators[id]
	return gen, ok
}


// Generate generates files for all enabled components.
// First calls platform generator to get modified CUE values and plaform deployment files, then calls component generators
// for components files.
func Generate(ctx *cue.Context, config cue.Value, plaformName string, enabledComponents map[string]bool) (map[string]map[string][]byte, error) {
	results := make(map[string]map[string][]byte)

	platGen, exists := NewPlatformRegistry().Get(generator.PlatformID(plaformName))
	if !exists {
		return nil, errors.New("failed to generate for unknown platform: " + plaformName)
	}

	// Call platform generator to generate platform deployment files
	modifiedConfig, platformFiles, err := platGen.Generate(ctx, config, enabledComponents)
	if err != nil {
		return nil, errors.New("failed to generate for platform " + plaformName + ": " + err.Error())
	}

	results[plaformName] = platformFiles

	components := NewComponentRegistry()

	// Now call component generators with the modified CUE configuration
	for componentName, isEnabled := range enabledComponents {
		if !isEnabled {
			continue
		}

		componentID := generator.ComponentID(componentName)
		componentGen, exists := components.Get(componentID)
		if !exists {
			// Skip if generator doesn't exist (platform-specific or optional component)
			continue
		}

		componentFiles, err := componentGen.GenerateComponent(modifiedConfig)
		if err != nil {
			return nil, errors.New("failed to generate component " + componentName + ": " + err.Error())
		}

		results[string(componentID)] = componentFiles
	}

	return results, nil
}
