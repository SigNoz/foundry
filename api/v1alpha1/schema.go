package v1alpha1

import (
	"embed"
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
)

var (
	//go:embed schema.json
	schema embed.FS

	// JSONSchema is the JSON schema for the API.
	jsonSchema *jsonschema.Resolved = mustNewJSONSchema()
)

func JSONSchema() *jsonschema.Resolved {
	return jsonSchema
}

func mustNewJSONSchema() *jsonschema.Resolved {
	contents, err := schema.ReadFile("schema.json")
	if err != nil {
		panic(err)
	}

	schema := new(jsonschema.Schema)
	if err := json.Unmarshal(contents, schema); err != nil {
		panic(err)
	}

	resolved, err := schema.Resolve(nil)
	if err != nil {
		panic(err)
	}

	return resolved
}
