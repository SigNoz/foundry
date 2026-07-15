package infrastructure

import "github.com/signoz/foundry/api/v1alpha1"

// Resource declares the resource this infrastructure is shaped for.
type Resource struct {
	// Kind of the resource this infrastructure serves.
	Kind ResourceKind `json:"kind,omitzero" yaml:"kind,omitempty" required:"true" description:"Kind of the resource this infrastructure serves" examples:"[\"Installation\"]"`

	// Specification for the resource.
	Spec ResourceSpec `json:"spec" yaml:"spec" required:"true" description:"Specification for the resource"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceSpec carries the identity of the casting embodying the resource.
type ResourceSpec struct {
	v1alpha1.TypeCastingRefSpec `json:",inline" yaml:",inline"`

	_ struct{} `additionalProperties:"false"`
}
