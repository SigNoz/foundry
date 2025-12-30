package signozotelcollector

import (
	"errors"
	"gopkg.in/yaml.v3"
	casting "github.com/SigNoz/foundry/internal/schema"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config casting.Config) (map[string][]byte, error) {
	files := make(map[string][]byte)

	collectorComponent, exists := config.Components["signozOtelCollector"]
	if !exists {
		return nil, errors.New("signozOtelCollector component not found in config")
	}

	configYAML, err := yaml.Marshal(collectorComponent["config"])
	if err != nil {
		return nil, errors.New("failed to marshal config: " + err.Error())
	}
	files["config.yaml"] = configYAML

	return files, nil
}
