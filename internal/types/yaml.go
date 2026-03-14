package types

import (
	"fmt"

	kyaml "sigs.k8s.io/yaml"
)

func UnmarshalYAML(data []byte, v any) error {
	return kyaml.Unmarshal(data, v)
}

func MustUnmarshalYAML(data []byte, v any) error {
	return kyaml.Unmarshal(data, v)
}

func MarshalYAML(v any) ([]byte, error) {
	yaml, err := kyaml.Marshal(v)
	if err != nil {
		return nil, err
	}

	return yaml, nil
}

func MustMarshalYAML(v any) []byte {
	yaml, err := MarshalYAML(v)
	if err != nil {
		panic(err)
	}

	return yaml
}

// MergeYAML takes a base YAML string and an override YAML string,
// and returns a new YAML string with deep merge — override keys win
// at every level, base-only keys are preserved.
func MergeYAML(base, override string) (string, error) {
	var baseMap map[string]any
	if err := kyaml.Unmarshal([]byte(base), &baseMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal base yaml: %w", err)
	}

	var overrideMap map[string]any
	if err := kyaml.Unmarshal([]byte(override), &overrideMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal override yaml: %w", err)
	}

	deepMerge(baseMap, overrideMap)

	merged, err := kyaml.Marshal(baseMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged yaml: %w", err)
	}

	return string(merged), nil
}

// deepMerge recursively merges override into base.
// For matching keys: if both values are maps, merge recursively.
// Otherwise, override wins.
func deepMerge(base, override map[string]any) {
	for k, overrideVal := range override {
		baseVal, exists := base[k]
		if !exists {
			base[k] = overrideVal
			continue
		}

		baseMap, baseIsMap := baseVal.(map[string]any)
		overrideMap, overrideIsMap := overrideVal.(map[string]any)

		if baseIsMap && overrideIsMap {
			deepMerge(baseMap, overrideMap)
		} else {
			base[k] = overrideVal
		}
	}
}
