package domain

import (
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

// MergeYAML deep-merges the override YAML onto the base YAML via RFC 7386 JSON
// Merge Patch: override wins at each leaf, base-only keys survive, lists are
// replaced wholesale, and a null in override deletes that key. It is
// StrategicMergeYAML with no declared list types.
func MergeYAML(base, override string) (string, error) {
	return StrategicMergeYAML(base, override, nil)
}
