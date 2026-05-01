package domain

import (
	"fmt"

	"github.com/tidwall/gjson"
)

type Material interface {
	Path() string
	FmtContents() []byte
}

type StructuredMaterial interface {
	Material
	Contents() []byte
	IsMultiDoc() bool
	WithContents(contents []byte) StructuredMaterial
	GetBytes(path string) ([]byte, error)
	GetStringSlice(path string) ([]string, error)
}

func getBytes(contents []byte, path string) ([]byte, error) {
	result := gjson.GetBytes(contents, path)
	if !result.Exists() {
		return nil, fmt.Errorf("path %q does not exist", path)
	}

	return []byte(result.String()), nil
}

func getStringSlice(contents []byte, path string) ([]string, error) {
	result := gjson.GetBytes(contents, path)
	if !result.Exists() {
		return nil, fmt.Errorf("path %q does not exist", path)
	}

	var items []string
	for _, item := range result.Array() {
		items = append(items, item.String())
	}

	return items, nil
}
