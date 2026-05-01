package domain

import (
	"github.com/signoz/foundry/internal/errors"
	"github.com/tidwall/gjson"
)

var (
	FormatYAML Format = Format{s: "yaml"}
	FormatJSON Format = Format{s: "json"}
	FormatINI  Format = Format{s: "ini"}
	FormatText Format = Format{s: "text"}
)

// Represents the format of a material's contents, which can be used to determine how to write it to disk or how to traverse and patch it.
type Format struct{ s string }

// Represents a material that foundry can write. It's typically a file, but could be something else in the future (e.g. a secret in a vault).
type Material interface {
	// Returns the output path for the material.
	Path() string

	// Returns the bytes in a format that should be written to disk. This is well indented and formatted for human readability.
	FmtContents() []byte
}

// Represents a Material whose contents can be traversed and patched.
type StructuredMaterial interface {
	Material

	// Returns the canonical JSON representation used for traversal and patching.
	JSONContents() []byte

	// IsMultiDoc reports whether the material represents multiple documents (e.g. multiple YAML documents separated by ---) or a single document. This is used to determine how to traverse and patch the material.
	IsMultiDoc() bool

	// Returns a copy of the material with replacement canonical JSON contents.
	CloneWithJSONContents(contents []byte) StructuredMaterial

	// Returns the value at the given path as bytes. Returns an error if the path does not exist.
	GetBytes(path string) ([]byte, error)

	// Returns the slice at the given path as strings. Returns an error if the path does not exist.
	GetStringSlice(path string) ([]string, error)
}

func getBytes(contents []byte, path string) ([]byte, error) {
	result := gjson.GetBytes(contents, path)
	if !result.Exists() {
		return nil, errors.Newf(errors.TypeNotFound, "path %q does not exist", path)
	}

	return []byte(result.String()), nil
}

func getStringSlice(contents []byte, path string) ([]string, error) {
	result := gjson.GetBytes(contents, path)
	if !result.Exists() {
		return nil, errors.Newf(errors.TypeNotFound, "path %q does not exist", path)
	}

	var items []string
	for _, item := range result.Array() {
		items = append(items, item.String())
	}

	return items, nil
}
