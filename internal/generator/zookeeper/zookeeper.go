package zookeeper

import (
	"errors"
	"gopkg.in/yaml.v3"
	casting "github.com/signoz/foundry/internal/schema"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config casting.Config) (map[string][]byte, error) {
	files := make(map[string][]byte)

	zookeeperComponent, exists := config.Components["zookeeper"]
	if !exists {
		return nil, errors.New("zookeeper component not found in config")
	}

	configBytes, err := yaml.Marshal(zookeeperComponent["config"])
	if err != nil {
		return nil, errors.New("failed to marshal config: " + err.Error())
	}
	files["zoo.cfg"] = configBytes

	return files, nil
}
