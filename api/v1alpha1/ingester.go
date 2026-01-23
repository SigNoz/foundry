package v1alpha1

import "github.com/signoz/foundry/internal/types"

var (
	// IngesterReceiverAddresses is the key for telemetry receiver endpoint addresses.
	IngesterReceiverAddresses string = "receiver"
)

type Ingester struct {
	Spec MoldingSpec `json:"spec" yaml:"spec"`

	Status MoldingStatus `json:"status" yaml:"status"`
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
