package domain

import (
	jsonpatchv5 "github.com/evanphx/json-patch/v5"
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

// MergeYAML deep-merges the override YAML onto the base YAML via RFC 7386 JSON
// Merge Patch: override wins at each leaf, base-only keys survive, lists are
// replaced wholesale, and a null in override deletes that key.
func MergeYAML(base, override string) (string, error) {
	baseJSON, err := kyaml.YAMLToJSON([]byte(base))
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to convert base yaml to json")
	}

	overrideJSON, err := kyaml.YAMLToJSON([]byte(override))
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInvalidInput, "failed to convert override yaml to json")
	}

	mergedJSON, err := jsonpatchv5.MergePatch(baseJSON, overrideJSON)
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to apply json merge patch")
	}

	merged, err := kyaml.JSONToYAML(mergedJSON)
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to convert merged json to yaml")
	}

	return string(merged), nil
}
