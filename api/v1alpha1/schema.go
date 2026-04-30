package v1alpha1

import (
	"embed"
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
	_ "github.com/google/jsonschema-go/jsonschema"
)

var (
	//go:embed schema.json
	schema embed.FS

	// JSONSchema is the JSON schema for the API.
	jsonSchema jsonschema.Schema = mustNewJSONSchema()
)

func JSONSchema() jsonschema.Schema {
	return jsonSchema
}

func mustNewJSONSchema() jsonschema.Schema {
	contents, err := schema.ReadFile("schema.json")
	if err != nil {
		panic(err)
	}

	var schema jsonschema.Schema
	if err := json.Unmarshal(contents, &schema); err != nil {
		panic(err)
	}

	return schema
}
