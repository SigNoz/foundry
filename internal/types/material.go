package types

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/util/yaml"
	kyaml "sigs.k8s.io/yaml"
)

type Material struct {
	contents []byte
	path     string
	format   Format
}

func NewMaterial(contents any, path string, format Format) (Material, error) {
	contentsBytes, err := json.Marshal(contents)
	if err != nil {
		return Material{}, fmt.Errorf("failed to marshal contents: %w", err)
	}

	return NewYAMLMaterial(contentsBytes, path)
}

func NewYAMLMaterial(contents []byte, path string) (Material, error) {
	jsonContents, err := yaml.ToJSON(contents)
	if err != nil {
		return Material{}, fmt.Errorf("invalid yaml: %w", err)
	}

	return Material{
		contents: jsonContents,
		path:     path,
		format:   FormatYAML,
	}, nil
}

func NewINIMaterial(contents []byte, path string) (Material, error) {
	jsonContents, err := INIToJSON(contents)
	if err != nil {
		return Material{}, fmt.Errorf("invalid ini: %w", err)
	}
	return Material{
		contents: jsonContents,
		path:     path,
		format:   FormatINI,
	}, nil
}

func (m Material) Contents() []byte {
	return m.contents
}

func (m Material) FmtContents() []byte {
	switch m.format {
	case FormatYAML:
		fmtContents, err := kyaml.JSONToYAML(m.contents)
		if err != nil {
			return nil
		}

		return fmtContents
	case FormatINI:
		fmtContents, err := JSONToINI(m.contents)
		if err != nil {
			return nil
		}
		return fmtContents
	default:
		return m.contents
	}
}

func (m Material) Path() string {
	return m.path
}

func (m Material) GetBytes(path string) ([]byte, error) {
	result := gjson.GetBytes(m.contents, path)
	if !result.Exists() {
		return nil, fmt.Errorf("path %q does not exist", path)
	}

	return []byte(result.String()), nil
}

func (m Material) GetStringSlice(path string) ([]string, error) {
	result := gjson.GetBytes(m.contents, path)

	if !result.Exists() {
		return nil, fmt.Errorf("path %q does not exist", path)
	}

	var items []string
	for _, item := range result.Array() {
		items = append(items, item.String())
	}

	return items, nil
}

func (m Material) ToYaml() ([]byte, error) {
	return kyaml.JSONToYAML(m.contents)
}
