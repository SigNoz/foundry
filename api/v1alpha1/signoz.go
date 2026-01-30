package v1alpha1

import "github.com/signoz/foundry/internal/types"

type SigNoz struct {
	// Specification for signoz.
	Spec MoldingSpec `json:"spec" yaml:"spec" jsonschema:"description=Specification for SigNoz"`

	// Status of signoz.
	Status SigNozStatus `json:"status" yaml:"status,omitempty" jsonschema:"description=Status of SigNoz"`
}

type SigNozStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	Addresses SigNozStatusAddresses `json:"addresses" yaml:"addresses,omitempty" jsonschema:"description=Addresses of SigNoz"`
}

type SigNozStatusAddresses struct {
	// API server addresses.
	APIServer []string `json:"apiserver" yaml:"apiserver" jsonschema:"description=API server addresses"`

	// Opamp server addresses.
	Opamp []string `json:"opamp" yaml:"opamp" jsonschema:"description=Opamp server addresses"`
}

func DefaultSigNoz() SigNoz {
	return SigNoz{
		Spec: MoldingSpec{
			Enabled: true,
			Cluster: TypeCluster{
				Replicas: types.NewIntPtr(1),
			},
			Version: "latest",
			Image:   "signoz/signoz:latest",
		},
	}
}
