package signoz

import (
	"fmt"
	"gopkg.in/yaml.v3"
	casting "github.com/SigNoz/foundry/internal/schema"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config casting.Config) (map[string][]byte, error) {
	files := make(map[string][]byte)

	signozComponent, exists := config.Components["signoz"]
	if !exists {
		return nil, fmt.Errorf("signoz component not found in config")
	}

	configYAML, err := yaml.Marshal(signozComponent["config"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	files["config.yaml"] = configYAML

	return files, nil
}