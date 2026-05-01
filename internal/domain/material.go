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

// Format identifies the syntax of a Material's contents.
type Format struct{ s string }

// Material is a unit of output that Foundry produces. It carries the path it
// should be written to and the bytes to write there.
type Material interface {
	Path() string

	// FmtContents returns the bytes in their human-readable, on-disk form. This
	// is the form Foundry writes out, distinct from the canonical form used for
	// traversal and patching.
	FmtContents() []byte
}

// StructuredMaterial is a Material whose contents are structured data with a
// navigable shape, supporting in-place reads and patches against a canonical
// representation.
type StructuredMaterial interface {
	Material

	// JSONContents returns the canonical JSON form used for traversal and
	// patching. JSON is the contract: callers (e.g. jsonpatch) operate on it
	// directly.
	JSONContents() []byte

	// HasMultipleDocuments reports whether the material groups multiple
	// top-level documents under one path (currently only multi-document YAML).
	// Callers use this to choose between scalar and array traversal of
	// JSONContents.
	HasMultipleDocuments() bool

	CloneWithJSONContents(contents []byte) StructuredMaterial

	// GetBytes returns the value at the given path as bytes. The path uses
	// gjson dotted-key syntax (e.g. "service.name", "service.names.0"), not
	// JSON Pointer.
	GetBytes(path string) ([]byte, error)

	// GetStringSlice returns the slice at the given path as strings. See
	// GetBytes for path syntax.
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
