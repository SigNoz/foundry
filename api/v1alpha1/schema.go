package v1alpha1

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
)

// MustResolveSchema parses and resolves a JSON Schema, panicking on failure.
func MustResolveSchema(bytes []byte) *jsonschema.Resolved {
	s := new(jsonschema.Schema)
	if err := json.Unmarshal(bytes, s); err != nil {
		panic(err)
	}
	resolved, err := s.Resolve(nil)
	if err != nil {
		panic(err)
	}
	return resolved
}
