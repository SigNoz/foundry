package infrastructure

import "github.com/signoz/foundry/api/v1alpha1"

// Resource declares the kind of resource this infrastructure is shaped for.
// It is a declaration, not a reference: the consumer owns the binding and
// declares it on its own casting.
type Resource struct {
	// Kind of the resource this infrastructure serves.
	Kind ResourceKind `json:"kind,omitzero" yaml:"kind,omitempty" required:"true" description:"Kind of the resource this infrastructure serves" examples:"[\"Installation\"]"`

	// Status of the resource.
	Status ResourceStatus `json:"status" yaml:"status,omitempty" description:"Status of the resource"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceStatus carries the requirement set a substrate shaped for the
// resource kind must satisfy.
type ResourceStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	// Addresses the resource admits at the substrate's edge.
	Addresses ResourceStatusAddresses `json:"addresses" yaml:"addresses,omitempty" description:"Addresses the resource admits at the substrate's edge"`

	_ struct{} `additionalProperties:"false"`
}

type ResourceStatusAddresses struct {
	// OTLP addresses.
	OTLP []string `json:"otlp" yaml:"otlp,omitempty" description:"OTLP addresses"`

	// API server addresses.
	APIServer []string `json:"apiserver" yaml:"apiserver,omitempty" description:"API server addresses"`

	_ struct{} `additionalProperties:"false"`
}
