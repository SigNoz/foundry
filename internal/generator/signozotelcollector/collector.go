package signozotelcollector

import (
	"errors"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/yaml"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config cue.Value) (map[string][]byte, error) {
	files := make(map[string][]byte)

	// Navigate to components.signozOtelCollector.config in the CUE value
	collectorConfig := config.LookupPath(cue.ParsePath("components.signozOtelCollector.config"))

	if collectorConfig.Exists() {
		// Export CUE value to YAML
		configYAML, err := yaml.Encode(collectorConfig)
		if err != nil {
			return nil, errors.New("failed to encode config: " + err.Error())
		}
		files["config.yaml"] = configYAML
	} else {
		return nil, errors.New("signozOtelCollector config not found in the provided CUE value")
	}

	return files, nil
}
