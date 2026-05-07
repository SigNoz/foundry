package domain

import (
	"encoding/json"

	"github.com/signoz/foundry/internal/errors"
)

var _ StructuredMaterial = JSONMaterial{}

type JSONMaterial struct {
	path     string
	contents []byte
}

func NewJSONMaterial(contents []byte, path string) (JSONMaterial, error) {
	if !json.Valid(contents) {
		return JSONMaterial{}, errors.Newf(errors.TypeInvalidInput, "failed to create JSON material for path %q: contents are not valid JSON", path)
	}

	return JSONMaterial{
		contents: contents,
		path:     path,
	}, nil
}

func MustNewJSONMaterial(contents []byte, path string) JSONMaterial {
	material, err := NewJSONMaterial(contents, path)
	if err != nil {
		panic(err)
	}

	return material
}

func (m JSONMaterial) Path() string {
	return m.path
}

func (m JSONMaterial) JSONContents() []byte {
	return m.contents
}

func (m JSONMaterial) HasMultipleDocuments() bool {
	return false
}

func (m JSONMaterial) FmtContents() []byte {
	return m.contents
}

func (m JSONMaterial) CloneWithJSONContents(contents []byte) StructuredMaterial {
	return JSONMaterial{
		contents: contents,
		path:     m.path,
	}
}

func (m JSONMaterial) GetBytes(path string) ([]byte, error) {
	return getBytes(m.contents, path)
}

func (m JSONMaterial) GetStringSlice(path string) ([]string, error) {
	return getStringSlice(m.contents, path)
}
