package v1alpha1

import "github.com/signoz/foundry/internal/types"

type Ingester struct {
	// Specification for the ingester.
	Spec MoldingSpec `json:"spec" yaml:"spec"`

	// Status of the ingester.
	Status IngesterStatus `json:"status" yaml:"status,omitempty"`
}

type IngesterStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	Addresses IngesterStatusAddresses `json:"addresses" yaml:"addresses,omitempty"`
}

type IngesterStatusAddresses struct {
	OTLP []string `json:"otlp" yaml:"otlp"`
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
