package signozotelcollector

import (
	"fmt"
	"gopkg.in/yaml.v3"
	casting "github.com/SigNoz/foundry/internal/schema"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config casting.Config) (map[string][]byte, error) {
	files := make(map[string][]byte)

	collectorComponent, exists := config.Components["signozOtelCollector"]
	if !exists {
		return nil, fmt.Errorf("signozOtelCollector component not found in config")
	}

	configYAML, err := yaml.Marshal(collectorComponent["config"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	files["config.yaml"] = configYAML

	return files, nil
}
