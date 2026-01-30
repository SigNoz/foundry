package v1alpha1

import "github.com/signoz/foundry/internal/types"

type Ingester struct {
	// Specification for the ingester.
	Spec MoldingSpec `json:"spec" yaml:"spec" jsonschema:"description=Specification for the ingester"`

	// Status of the ingester.
	Status IngesterStatus `json:"status" yaml:"status,omitempty" jsonschema:"description=Status of the ingester"`
}

type IngesterStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	Addresses IngesterStatusAddresses `json:"addresses" yaml:"addresses,omitempty" jsonschema:"description=Addresses of the ingester"`
}

type IngesterStatusAddresses struct {
	OTLP []string `json:"otlp" yaml:"otlp" jsonschema:"description=OTLP addresses"`
}

func DefaultIngester() Ingester {
	return Ingester{
		Spec: MoldingSpec{
			Enabled: true,
			Cluster: TypeCluster{
				Replicas: types.NewIntPtr(1),
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
