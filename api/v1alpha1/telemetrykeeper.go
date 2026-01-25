package v1alpha1

import (
	"github.com/signoz/foundry/internal/types"
)

type TelemetryKeeper struct {
	// Kind of the telemetry keeper to use.
	Kind TelemetryKeeperKind `json:"kind,omitzero" yaml:"kind,omitempty"`

	// Specification for the telemetry keeper.
	Spec MoldingSpec `json:"spec" yaml:"spec"`

	// Status of the telemetry keeper.
	Status TelemetryKeeperStatus `json:"status" yaml:"status,omitempty"`
}

type TelemetryKeeperStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	// Addresses of the telemetry keeper.
	Addresses TelemetryKeeperStatusAddresses `json:"addresses" yaml:"addresses,omitempty"`
}

type TelemetryKeeperStatusAddresses struct {
	// Raft addresses.
	Raft []string `json:"raft" yaml:"raft,omitempty"`

	// Client addresses.
	Client []string `json:"client" yaml:"client,omitempty"`
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
