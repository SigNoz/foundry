package registry

import (
	"github.com/SigNoz/foundry/internal/loader"
	"github.com/SigNoz/foundry/internal/schema"
)

type Factory struct {
	castingConfig     casting.Config
	enabledComponents map[string]bool
}

func NewFactory(loadedConfig *loader.LoadedConfig) (*Factory, error) {
	var config casting.Config
	err := loadedConfig.Unified.Decode(&config)
	if err != nil{
		return &Factory{}, err

	}

	return &Factory{
		castingConfig:     config,
		enabledComponents: loadedConfig.EnabledComponents,
	}, nil
}

func (f *Factory) CreateComponentRegistry() *ComponentRegistry {
	return NewComponentRegistry(f.castingConfig, f.enabledComponents)
}

func (f *Factory) CreateOrchestratorRegistry() *OrchestratorRegistry {
	return NewOrchestratorRegistry(f.castingConfig)
}

func (f *Factory) CreateBoth() (*ComponentRegistry, *OrchestratorRegistry) {
	return f.CreateComponentRegistry(), f.CreateOrchestratorRegistry()
}
