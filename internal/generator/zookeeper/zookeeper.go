package zookeeper

import (
	"fmt"
	"gopkg.in/yaml.v3"
	casting "github.com/SigNoz/foundry/internal/schema"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config casting.Config) (map[string][]byte, error) {
	files := make(map[string][]byte)

	zookeeperComponent, exists := config.Components["zookeeper"]
	if !exists {
		return nil, fmt.Errorf("zookeeper component not found in config")
	}

	configBytes, err := yaml.Marshal(zookeeperComponent["config"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	files["zoo.cfg"] = configBytes

	return files, nil
}
