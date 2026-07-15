package infrastructure

import "github.com/signoz/foundry/api/v1alpha1"

// Resource is the resource this infrastructure serves.
type Resource struct {
	// Kind of the resource this infrastructure serves.
	Kind ResourceKind `json:"kind,omitzero" yaml:"kind,omitempty" required:"true" description:"Kind of the resource this infrastructure serves" examples:"[\"Installation\"]"`

	// Specification for the resource.
	Spec ResourceSpec `json:"spec" yaml:"spec" required:"true" description:"Specification for the resource"`

	// Status of the resource.
	Status ResourceStatus `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the resource"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceSpec carries the identity of the casting embodying the resource.
type ResourceSpec struct {
	v1alpha1.TypeCastingRef `json:",inline" yaml:",inline"`

	_ struct{} `additionalProperties:"false"`
}

type ResourceStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	Resolution ResourceStatusResolution `json:"resolution,omitzero" yaml:"resolution,omitempty" description:"The casting the resource resolved to"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceStatusResolution records the casting the resource resolved to.
type ResourceStatusResolution struct {
	Checksum string   `json:"checksum,omitempty" yaml:"checksum,omitempty" description:"Checksum of the resolved casting"`
	Casting  string   `json:"casting,omitempty" yaml:"casting,omitempty" description:"The resolved casting, verbatim"`
	_        struct{} `additionalProperties:"false"`
}
