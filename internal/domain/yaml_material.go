package domain

import (
	"bytes"
	"encoding/json"

	"github.com/signoz/foundry/internal/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	kyaml "sigs.k8s.io/yaml"
)

var _ StructuredMaterial = YAMLMaterial{}

type YAMLMaterial struct {
	path                 string
	contents             []byte
	hasMultipleDocuments bool
}

func NewYAMLMaterial(contents []byte, path string) (YAMLMaterial, error) {
	reader := kio.ByteReader{Reader: bytes.NewReader(contents), OmitReaderAnnotations: true}

	nodes, err := reader.Read()
	if err != nil {
		return YAMLMaterial{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create YAML material for path %q: contents are not valid YAML", path)
	}

	var jsonContents []byte
	if len(nodes) == 1 {
		jsonContents, err = nodes[0].MarshalJSON()
		if err != nil {
			return YAMLMaterial{}, errors.Wrapf(err, errors.TypeInternal, "failed to convert YAML material to canonical JSON for path %q", path)
		}

		return YAMLMaterial{
			contents:             jsonContents,
			path:                 path,
			hasMultipleDocuments: false,
		}, nil
	}

	var docs []json.RawMessage
	for _, node := range nodes {
		jsonContents, err := node.MarshalJSON()
		if err != nil {
			return YAMLMaterial{}, errors.Wrapf(err, errors.TypeInternal, "failed to convert YAML material to canonical JSON for path %q", path)
		}

		docs = append(docs, jsonContents)
	}

	jsonContents, err = json.Marshal(docs)
	if err != nil {
		return YAMLMaterial{}, errors.Wrapf(err, errors.TypeInternal, "failed to convert multi-document YAML material to canonical JSON for path %q", path)
	}

	return YAMLMaterial{
		contents:             jsonContents,
		path:                 path,
		hasMultipleDocuments: len(nodes) > 1,
	}, nil
}

func MustNewYAMLMaterial(contents []byte, path string) YAMLMaterial {
	material, err := NewYAMLMaterial(contents, path)
	if err != nil {
		panic(err)
	}

	return material
}

func (m YAMLMaterial) Path() string {
	return m.path
}

func (m YAMLMaterial) JSONContents() []byte {
	return m.contents
}

func (m YAMLMaterial) HasMultipleDocuments() bool {
	return m.hasMultipleDocuments
}

func (m YAMLMaterial) FmtContents() []byte {
	if !m.HasMultipleDocuments() {
		node, err := kyaml.JSONToYAML(m.contents)
		if err != nil {
			return nil
		}

		return node
	}

	var docs []json.RawMessage
	if err := json.Unmarshal(m.contents, &docs); err != nil {
		return nil
	}

	var nodes []*yaml.RNode
	for _, doc := range docs {
		node, err := yaml.ConvertJSONToYamlNode(string(doc))
		if err != nil {
			return nil
		}

		nodes = append(nodes, node)
	}

	var buf bytes.Buffer
	writer := kio.ByteWriter{Writer: &buf, KeepReaderAnnotations: true}

	if err := writer.Write(nodes); err != nil {
		return nil
	}

	return buf.Bytes()
}

func (m YAMLMaterial) CloneWithJSONContents(contents []byte) StructuredMaterial {
	return YAMLMaterial{
		contents:             contents,
		path:                 m.path,
		hasMultipleDocuments: m.hasMultipleDocuments,
	}
}

func (m YAMLMaterial) GetBytes(path string) ([]byte, error) {
	return getBytes(m.contents, path)
}

func (m YAMLMaterial) GetStringSlice(path string) ([]string, error) {
	return getStringSlice(m.contents, path)
}
