package v1alpha1

// CastingMeta carries the apiVersion, kind, metadata, and status fields
// shared by every casting kind. Per-Kind Casting structs embed it inline.
type CastingMeta struct {
	TypeVersion `json:",inline" yaml:",inline"`
	Kind        Kind         `json:"kind" yaml:"kind" required:"true" description:"Kind of the casting resource."`
	Metadata    TypeMetadata `json:"metadata" yaml:"metadata" required:"true" description:"Metadata of the casting configuration"`
	Status      Status       `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the casting"`
	_           struct{}     `additionalProperties:"false"`
}

// Status carries the casting file's checksum.
type Status struct {
	Checksum string   `json:"checksum" yaml:"checksum" description:"Checksum of the casting file"`
	_        struct{} `additionalProperties:"false"`
}
