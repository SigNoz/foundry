package v1alpha1

import "github.com/signoz/foundry/internal/types"

var (
	// SignozAPIAddresses is the key for API endpoint addresses.
	SignozAPIAddresses string = "api"
)

type SigNoz struct {
	Spec MoldingSpec `json:"spec" yaml:"spec"`

	Status MoldingStatus `json:"status" yaml:"status"`
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
