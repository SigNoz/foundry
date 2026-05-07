package v1alpha1

type SigNoz struct {
	// Specification for signoz.
	Spec MoldingSpec `json:"spec" yaml:"spec" jsonschema:"description=Specification for SigNoz"`

	// Status of signoz.
	Status SigNozStatus `json:"status" yaml:"status,omitempty" jsonschema:"description=Status of SigNoz"`

	_ struct{} `additionalProperties:"false"`
}

type SigNozStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	Addresses SigNozStatusAddresses `json:"addresses" yaml:"addresses,omitempty" jsonschema:"description=Addresses of SigNoz"`

	_ struct{} `additionalProperties:"false"`
}

type SigNozStatusAddresses struct {
	// API server addresses.
	APIServer []string `json:"apiserver" yaml:"apiserver" jsonschema:"description=API server addresses"`

	// Opamp server addresses.
	Opamp []string `json:"opamp" yaml:"opamp" jsonschema:"description=Opamp server addresses"`

	_ struct{} `additionalProperties:"false"`
}

func DefaultSigNoz() SigNoz {
	return SigNoz{
		Spec: MoldingSpec{
			Enabled: boolPtr(true),
			Cluster: TypeCluster{
				Replicas: intPtr(1),
			},
			Version: "latest",
			Image:   "signoz/signoz:latest",
		},
	}
}
