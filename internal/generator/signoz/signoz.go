package signoz

import (
	"errors"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/yaml"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config cue.Value) (map[string][]byte, error) {
	files := make(map[string][]byte)

	// Navigate to components.signoz.config in the CUE value
	signozConfig := config.LookupPath(cue.ParsePath("components.signoz.config"))
	if !signozConfig.Exists() {
		// Config is optional - generate minimal default
		files["config.yaml"] = []byte("{}\n")
		return files, nil
	}

	// Export CUE value to YAML
	configYAML, err := yaml.Encode(signozConfig)
	if err != nil {
		return nil, errors.New("failed to encode config: " + err.Error())
	}
	files["config.yaml"] = configYAML

	return files, nil
}