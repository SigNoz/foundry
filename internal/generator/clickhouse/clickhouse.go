package clickhouse

import (
	"errors"
	"fmt"

	"cuelang.org/go/cue"
	"gopkg.in/yaml.v3"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config cue.Value) (map[string][]byte, error) {
	files := make(map[string][]byte)

	// Navigate to components.clickhouse.config in the CUE value
	clickhouseConfig := config.LookupPath(cue.ParsePath("components.clickhouse.config"))
	// Decode to map for processing multiple files
	var componentConfig map[string]any
	if err := clickhouseConfig.Decode(&componentConfig); err != nil {
		return nil, errors.New("failed to decode clickhouse config: " + err.Error())
	}

	// Generate config.yaml (main ClickHouse configuration)
	if serverConfig, exists := componentConfig["serverConfig"]; exists {
		configYAML, err := yaml.Marshal(serverConfig)
		if err != nil {
			return nil, errors.New("failed to marshal serverConfig: " + err.Error())
		}
		files["config.yaml"] = configYAML
	}

	// Generate users.yaml (users, profiles, quotas)
	if usersConfig, exists := componentConfig["usersConfig"]; exists {
		usersYAML, err := yaml.Marshal(usersConfig)
		if err != nil {
			return nil, errors.New("failed to marshal usersConfig: " + err.Error())
		}
		files["users.yaml"] = usersYAML
	}

	// Generate custom-function.yaml
	if customFnConfig, exists := componentConfig["customFunctionConfig"]; exists {
		customFnYAML, err := yaml.Marshal(customFnConfig)
		if err != nil {
			return nil, errors.New("failed to marshal customFunctionConfig: " + err.Error())
		}
		files["custom-function.yaml"] = customFnYAML
	}

	// Generate additional config files in config.d/
	if configD, exists := componentConfig["config_d"]; exists {
		if configFiles, ok := configD.(map[string]any); ok {
			for filename, content := range configFiles {
				fileYAML, err := yaml.Marshal(content)
				if err != nil {
					return nil, errors.New("failed to marshal config_d/" + filename + ": " + err.Error())
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
					return nil, errors.New("failed to marshal users_d/" + filename + ": " + err.Error())
				}
				files[fmt.Sprintf("users.d/%s.yaml", filename)] = fileYAML
			}
		}
	}

	return files, nil
}
