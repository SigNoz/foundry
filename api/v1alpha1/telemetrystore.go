package v1alpha1

import (
	"github.com/signoz/foundry/internal/types"
)

var (
	// TelemetryStoreClusterAddresses is the key for cluster node addresses.
	TelemetryStoreClusterAddresses string = "cluster"
)

type TelemetryStore struct {
	// Kind of the telemetry store to use.
	Kind TelemetryStoreKind `json:"kind,omitzero" yaml:"kind,omitempty"`

	// Specification for the telemetry store.
	Spec MoldingSpec `json:"spec" yaml:"spec"`

	// Status of the telemetry store.
	Status MoldingStatus `json:"status" yaml:"status"`
}

func DefaultTelemetryStore() TelemetryStore {
	return TelemetryStore{
		Kind: TelemetryStoreKindClickhouse,
		Spec: MoldingSpec{
			Enabled: true,
			Cluster: TypeCluster{
				Replicas: types.NewIntPtr(0),
				Shards:   types.NewIntPtr(1),
			},
			Version: "25.5.6",
			Image:   "clickhouse/clickhouse-server:25.5.6",
		},
	}
}
