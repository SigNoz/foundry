package domain

import (
	"encoding/json"
	"fmt"
)

var _ StructuredMaterial = JSONMaterial{}

type JSONMaterial struct {
	structuredData
}

func NewJSONMaterial(contents []byte, path string) (JSONMaterial, error) {
	if !json.Valid(contents) {
		return JSONMaterial{}, fmt.Errorf("invalid json for %s", path)
	}

	return JSONMaterial{
		structuredData: structuredData{
			contents: contents,
			path:     path,
		},
	}, nil
}

func (m JSONMaterial) IsMultiDoc() bool {
	return false
}

func (m JSONMaterial) FmtContents() []byte {
	return m.contents
}

func (m JSONMaterial) WithContents(contents []byte) StructuredMaterial {
	return JSONMaterial{
		structuredData: structuredData{
			contents: contents,
			path:     m.path,
		},
	}
}
