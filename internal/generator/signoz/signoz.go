package signoz

import (
	"errors"
	"gopkg.in/yaml.v3"
	casting "github.com/SigNoz/foundry/internal/schema"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config casting.Config) (map[string][]byte, error) {
	files := make(map[string][]byte)

	signozComponent, exists := config.Components["signoz"]
	if !exists {
		return nil, errors.New("signoz component not found in config")
	}

	configYAML, err := yaml.Marshal(signozComponent["config"])
	if err != nil {
		return nil, errors.New("failed to marshal config: " + err.Error())
	}
	files["config.yaml"] = configYAML

	return files, nil
}