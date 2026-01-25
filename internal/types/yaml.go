package types

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
