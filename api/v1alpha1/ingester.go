package v1alpha1

import "github.com/signoz/foundry/internal/types"

type Ingester struct {
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
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
