package domain

import (
	"github.com/signoz/foundry/internal/errors"
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

// MergeYAML deep-merges the override YAML string onto the base YAML string,
// applying the per-path list strategy (see Merge; nil = a plain deep-merge with
// lists replaced), and returns the result.
func MergeYAML(base, override string, strategy map[string]ListMerge) (string, error) {
	var baseMap map[string]any
	if err := kyaml.Unmarshal([]byte(base), &baseMap); err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to unmarshal base yaml")
	}

	var overrideMap map[string]any
	if err := kyaml.Unmarshal([]byte(override), &overrideMap); err != nil {
		return "", errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal override yaml")
	}

	Merge(baseMap, overrideMap, strategy)

	merged, err := kyaml.Marshal(baseMap)
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to marshal merged yaml")
	}

	return string(merged), nil
}
