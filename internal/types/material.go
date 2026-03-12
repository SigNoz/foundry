package types

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
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

func NewTextMaterial(contents []byte, path string) Material {
	return Material{
		contents: contents,
		path:     path,
		format:   FormatText,
	}
}

func NewYAMLMaterial(contents []byte, path string) (Material, error) {
	nodes, err := (&kio.ByteReader{
		Reader:                bytes.NewReader(contents),
		OmitReaderAnnotations: true,
	}).Read()
	if err != nil {
		return Material{}, fmt.Errorf("invalid yaml: %w", err)
	}

	var jsonContents []byte
	if len(nodes) == 1 {
		jsonContents, err = nodes[0].MarshalJSON()
		if err != nil {
			return Material{}, fmt.Errorf("failed to marshal node to json: %w", err)
		}
	} else {
		var docs []json.RawMessage
		for _, node := range nodes {
			j, err := node.MarshalJSON()
			if err != nil {
				return Material{}, fmt.Errorf("failed to marshal node to json: %w", err)
			}
			docs = append(docs, j)
		}
		jsonContents, err = json.Marshal(docs)
		if err != nil {
			return Material{}, fmt.Errorf("failed to marshal docs to json array: %w", err)
		}
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

func (m Material) IsMultiDoc() bool {
	return m.format == FormatYAML && gjson.ParseBytes(m.contents).IsArray()
}

func (m Material) FmtContents() []byte {
	switch m.format {
	case FormatYAML:
		out, err := m.ToYaml()
		if err != nil {
			return nil
		}
		return out
	case FormatINI:
		fmtContents, err := JSONToINI(m.contents)
		if err != nil {
			return nil
		}
		return fmtContents
	case FormatText:
		return m.contents
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
	if !m.IsMultiDoc() {
		node, err := yaml.ConvertJSONToYamlNode(string(m.contents))
		if err != nil {
			return nil, err
		}
		return []byte(node.MustString()), nil
	}

	var docs []json.RawMessage
	if err := json.Unmarshal(m.contents, &docs); err != nil {
		return nil, err
	}

	var nodes []*yaml.RNode
	for _, doc := range docs {
		node, err := yaml.ConvertJSONToYamlNode(string(doc))
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	var buf bytes.Buffer
	err := (&kio.ByteWriter{
		Writer:                &buf,
		KeepReaderAnnotations: true,
	}).Write(nodes)
	return buf.Bytes(), err
}
