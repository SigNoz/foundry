package infrastructure

import "github.com/signoz/foundry/api/v1alpha1"

// Resource is the resource molding's slot on an Infrastructure casting: what a
// substrate must provide, and what foundry derived from it.
type Resource struct {
	// Specification for the resource.
	Spec v1alpha1.MoldingSpec `json:"spec" yaml:"spec" jsonschema:"description=Specification for the resource"`

	// Status of the resource.
	Status ResourceStatus `json:"status" yaml:"status,omitempty" description:"Status of the resource"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceStatus carries the settled requirement document.
type ResourceStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	_ struct{} `additionalProperties:"false"`
}
