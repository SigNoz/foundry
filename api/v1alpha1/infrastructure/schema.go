package infrastructure

import (
	_ "embed"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/signoz/foundry/api/v1alpha1"
)

//go:embed casting.schema.json
var schemaJSON []byte

var schema = v1alpha1.MustResolveSchema(schemaJSON)

// Schema returns the resolved JSON schema for an Infrastructure casting.
func Schema() *jsonschema.Resolved {
	return schema
}
