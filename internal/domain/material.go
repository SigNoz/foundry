package domain

import (
	"github.com/signoz/foundry/internal/errors"
	"github.com/tidwall/gjson"
)

var (
	FormatYAML = Format{s: "yaml", new: func(c []byte, p string) (Material, error) { return NewYAMLMaterial(c, p) }}
	FormatJSON = Format{s: "json", new: func(c []byte, p string) (Material, error) { return NewJSONMaterial(c, p) }}
	FormatINI  = Format{s: "ini", new: func(c []byte, p string) (Material, error) { return NewINIMaterial(c, p) }}
	FormatText = Format{s: "text", new: func(c []byte, p string) (Material, error) { return NewBlobMaterial(c, p), nil }}
)

// Format identifies the syntax of a Material's contents and carries the
// constructor that wraps a rendered byte stream in the matching concrete
// Material type. To register a new format, add a single entry above with its
// New* constructor — Template.Render picks it up automatically.
type Format struct {
	s   string
	new func(contents []byte, path string) (Material, error)
}

func (f Format) String() string {
	return f.s
}

// NewMaterial wraps contents in the concrete Material type for this format.
func (f Format) NewMaterial(contents []byte, path string) (Material, error) {
	if f.new == nil {
		return nil, errors.Newf(errors.TypeUnsupported, "failed to create material for path %q: unsupported format %q", path, f.s)
	}
	return f.new(contents, path)
}

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
