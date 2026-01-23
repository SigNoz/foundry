package v1alpha1

import (
	"github.com/signoz/foundry/internal/types"
)

var (
	// TelemetryKeeperRaftAddresses is the key for inter-keeper consensus coordination.
	TelemetryKeeperRaftAddresses string = "raft"
	// TelemetryKeeperClientAddresses is the key for client connections.
	TelemetryKeeperClientAddresses string = "client"
)

type TelemetryKeeper struct {
	// Kind of the telemetry keeper to use.
	Kind TelemetryKeeperKind `json:"kind,omitzero" yaml:"kind,omitempty"`

	// Specification for the telemetry keeper.
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func DefaultTelemetryKeeper() TelemetryKeeper {
	return TelemetryKeeper{
		Kind: TelemetryKeeperKindClickhouseKeeper,
		Spec: MoldingSpec{
			Enabled: true,
			Cluster: TypeCluster{
				Replicas: types.NewIntPtr(1),
			},
			Version: "25.5.6",
			Image:   "clickhouse/clickhouse-keeper:25.5.6",
		},
	}
}
