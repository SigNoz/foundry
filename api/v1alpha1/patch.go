package v1alpha1

const (
	// PatchTypeJSONPatch is the default patch type using JSON Patch (RFC 6902).
	PatchTypeJSONPatch = "jsonpatch"
)

// PatchEntry is a set of patch operations targeting a specific generated file.
type PatchEntry struct {
	// Type selects the patch driver. Defaults to "jsonpatch" if empty.
	Type string `json:"type,omitempty" yaml:"type,omitempty" enum:",jsonpatch" description:"Patch driver type. Defaults to jsonpatch." default:"jsonpatch" example:"jsonpatch"`

	// Target is the output file to patch, relative to the pours directory.
	Target string `json:"target" yaml:"target" required:"true" minLength:"1" description:"Target output file to patch" examples:"[\"compose.yaml\",\"signoz/deployment.yaml\",\"values.yaml\",\"telemetrystore/telemtrystore-clickhouse-0-*.yaml\"]"`

	// Operations is a list of JSON Patch (RFC 6902) operations to apply. Used by the jsonpatch driver.
	Operations []PatchOperation `json:"operations" yaml:"operations" required:"true" minItems:"1" description:"JSON Patch (RFC 6902) operations to apply. Used by the jsonpatch driver."`

	_ struct{} `additionalProperties:"false"`
}

// PatchType returns the patch type, defaulting to PatchTypeJSONPatch if empty.
func (pe PatchEntry) PatchType() string {
	if pe.Type == "" {
		return PatchTypeJSONPatch
	}

	return pe.Type
}

// PatchOperation is a single JSON Patch (RFC 6902) operation. Used by the jsonpatch driver.
type PatchOperation struct {
	// Op is the JSON Patch (RFC 6902) operation type: add, remove, replace, move, copy, test.
	Op string `json:"op" yaml:"op" required:"true" enum:"add,remove,replace,move,copy,test" description:"JSON Patch (RFC 6902) operation type"`

	// Path is a JSON Pointer (RFC 6902) to the target location.
	Path string `json:"path" yaml:"path" required:"true" pattern:"^/" description:"JSON Pointer (RFC 6901) to the target location" example:"/services/clickhouse/mem_limit"`

	// Value is the value for add, replace, or test operations.
	Value any `json:"value,omitempty" yaml:"value,omitempty" description:"Value for add, replace, or test operations"`

	// From is a JSON Pointer for the source location in move and copy operations.
	From string `json:"from,omitempty" yaml:"from,omitempty" pattern:"^/" description:"Source JSON Pointer for move and copy operations" example:"/services/clickhouse/old_field"`

	_ struct{} `additionalProperties:"false"`
}
