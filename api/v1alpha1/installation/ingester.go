package installation

import "github.com/signoz/foundry/api/v1alpha1"

type Ingester struct {
	// Specification for the ingester.
	Spec v1alpha1.MoldingSpec `json:"spec" yaml:"spec" jsonschema:"description=Specification for the ingester"`

	// Status of the ingester.
	Status IngesterStatus `json:"status" yaml:"status,omitempty" jsonschema:"description=Status of the ingester"`

	_ struct{} `additionalProperties:"false"`
}

type IngesterStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	Addresses IngesterStatusAddresses `json:"addresses" yaml:"addresses,omitempty" jsonschema:"description=Addresses of the ingester"`

	_ struct{} `additionalProperties:"false"`
}

type IngesterStatusAddresses struct {
	OTLP []string `json:"otlp" yaml:"otlp" jsonschema:"description=OTLP addresses"`

	_ struct{} `additionalProperties:"false"`
}

func DefaultIngester() Ingester {
	return Ingester{
		Spec: v1alpha1.MoldingSpec{
			Enabled: v1alpha1.BoolPtr(true),
			Cluster: v1alpha1.TypeCluster{
				Replicas: v1alpha1.IntPtr(1),
			},
			Version: "latest",
			Image:   "signoz/signoz-otel-collector:latest",
			Env:     map[string]string{},
			Config: v1alpha1.TypeConfig{
				Data: map[string]string{},
			},
		},
	}
}
