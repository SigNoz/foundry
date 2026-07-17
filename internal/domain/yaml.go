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

// MergeYAMLWithStrategy deep-merges the override YAML onto the base YAML using
// the per-path list strategy (see Merge). It is the list-aware counterpart to
// MergeYAML, for configs whose lists must union or keep order rather than be
// replaced wholesale (e.g. an OTel collector's pipeline receivers/processors).
func MergeYAMLWithStrategy(base, override string, strategy map[string]ListMerge) (string, error) {
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
