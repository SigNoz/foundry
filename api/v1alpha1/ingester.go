package v1alpha1

type Ingester struct {
	// Specification for the ingester.
	Spec MoldingSpec `json:"spec" yaml:"spec" jsonschema:"description=Specification for the ingester"`

	// Status of the ingester.
	Status IngesterStatus `json:"status" yaml:"status,omitempty" jsonschema:"description=Status of the ingester"`

	_ struct{} `additionalProperties:"false"`
}

type IngesterStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	Addresses IngesterStatusAddresses `json:"addresses" yaml:"addresses,omitempty" jsonschema:"description=Addresses of the ingester"`

	_ struct{} `additionalProperties:"false"`
}

type IngesterStatusAddresses struct {
	OTLP []string `json:"otlp" yaml:"otlp" jsonschema:"description=OTLP addresses"`

	_ struct{} `additionalProperties:"false"`
}

func DefaultIngester() Ingester {
	return Ingester{
		Spec: MoldingSpec{
			Enabled: boolPtr(true),
			Cluster: TypeCluster{
				Replicas: intPtr(1),
			},
			Version: "latest",
			Image:   "signoz/signoz-otel-collector:latest",
			Env:     map[string]string{},
			Config: TypeConfig{
				Data: map[string]string{},
			},
		},
	}
}
