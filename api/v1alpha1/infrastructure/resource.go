package infrastructure

import "github.com/signoz/foundry/api/v1alpha1"

// Resource is the infrastructure kind's molding: the unit to be hosted. Its
// kinds are the consumer casting kinds.
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

	Addresses ResourceStatusAddresses `json:"addresses,omitzero" yaml:"addresses,omitempty" description:"Addresses the resource exposes"`

	_ struct{} `additionalProperties:"false"`
}

type ResourceStatusAddresses struct {
	OTLP []string `json:"otlp,omitempty" yaml:"otlp,omitempty" description:"OTLP addresses"`

	UI []string `json:"ui,omitempty" yaml:"ui,omitempty" description:"UI addresses"`

	_ struct{} `additionalProperties:"false"`
}
