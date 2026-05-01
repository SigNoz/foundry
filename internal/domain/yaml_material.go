package domain

import (
	"bytes"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	kyaml "sigs.k8s.io/yaml"
)

var _ StructuredMaterial = YAMLMaterial{}

type YAMLMaterial struct {
	structuredData
	multiDoc bool
}

func NewYAMLMaterial(contents []byte, path string) (YAMLMaterial, error) {
	nodes, err := (&kio.ByteReader{
		Reader:                bytes.NewReader(contents),
		OmitReaderAnnotations: true,
	}).Read()
	if err != nil {
		return YAMLMaterial{}, fmt.Errorf("invalid yaml: %w", err)
	}

	var jsonContents []byte
	if len(nodes) == 1 {
		jsonContents, err = nodes[0].MarshalJSON()
		if err != nil {
			return YAMLMaterial{}, fmt.Errorf("failed to marshal node to json: %w", err)
		}
	} else {
		var docs []json.RawMessage
		for _, node := range nodes {
			j, err := node.MarshalJSON()
			if err != nil {
				return YAMLMaterial{}, fmt.Errorf("failed to marshal node to json: %w", err)
			}
			docs = append(docs, j)
		}
		jsonContents, err = json.Marshal(docs)
		if err != nil {
			return YAMLMaterial{}, fmt.Errorf("failed to marshal docs to json array: %w", err)
		}
	}

	return YAMLMaterial{
		structuredData: structuredData{
			contents: jsonContents,
			path:     path,
		},
		multiDoc: len(nodes) > 1,
	}, nil
}

func (m YAMLMaterial) IsMultiDoc() bool {
	return m.multiDoc
}

func (m YAMLMaterial) FmtContents() []byte {
	fmtContents, err := m.toYAML()
	if err != nil {
		return nil
	}
	return fmtContents
}

func (m YAMLMaterial) WithContents(contents []byte) StructuredMaterial {
	return YAMLMaterial{
		structuredData: structuredData{
			contents: contents,
			path:     m.path,
		},
		multiDoc: m.multiDoc,
	}
}

func (m YAMLMaterial) toYAML() ([]byte, error) {
	if !m.IsMultiDoc() {
		node, err := kyaml.JSONToYAML(m.contents)
		if err != nil {
			return nil, err
		}
		return node, nil
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
