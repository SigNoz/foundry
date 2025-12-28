package clickhouse

import (
	"fmt"
	"gopkg.in/yaml.v3"
	casting "github.com/SigNoz/foundry/internal/schema"
)

type Generator struct{}


func (g *Generator) GenerateComponent(config casting.Config) (map[string][]byte, error) {
	files := make(map[string][]byte)

	clickhouseComponent, exists := config.Components["clickhouse"]
	if !exists {
		return nil, fmt.Errorf("clickhouse component not found in config")
	}

	componentConfig, ok := clickhouseComponent["config"].(map[string]any)
	if !ok {
		var configBytes []byte
		configBytes, ok = clickhouseComponent["config"].([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid component config format")
		}
		err := yaml.Unmarshal(configBytes, &componentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal component config: %w", err)
		}
	}

	// Generate config.yaml (main ClickHouse configuration)
	if serverConfig, exists := componentConfig["serverConfig"]; exists {
		configYAML, err := yaml.Marshal(serverConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal serverConfig: %w", err)
		}
		files["config.yaml"] = configYAML
	}

	// Generate users.yaml (users, profiles, quotas)
	if usersConfig, exists := componentConfig["usersConfig"]; exists {
		usersYAML, err := yaml.Marshal(usersConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal usersConfig: %w", err)
		}
		files["users.yaml"] = usersYAML
	}

	// Generate custom-function.yaml
	if customFnConfig, exists := componentConfig["customFunctionConfig"]; exists {
		customFnYAML, err := yaml.Marshal(customFnConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal customFunctionConfig: %w", err)
		}
		files["custom-function.yaml"] = customFnYAML
	}

	// Generate additional config files in config.d/
	if configD, exists := componentConfig["config_d"]; exists {
		if configFiles, ok := configD.(map[string]any); ok {
			for filename, content := range configFiles {
				fileYAML, err := yaml.Marshal(content)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal config_d/%s: %w", filename, err)
				}
				files[fmt.Sprintf("config.d/%s.yaml", filename)] = fileYAML
			}
		}
	}

	// Generate additional user files in users.d/
	if usersD, exists := componentConfig["users_d"]; exists {
		if userFiles, ok := usersD.(map[string]any); ok {
			for filename, content := range userFiles {
				fileYAML, err := yaml.Marshal(content)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal users_d/%s: %w", filename, err)
				}
				files[fmt.Sprintf("users.d/%s.yaml", filename)] = fileYAML
			}
		}
	}

	return files, nil
}
