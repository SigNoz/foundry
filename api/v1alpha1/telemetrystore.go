package v1alpha1

import (
	"github.com/signoz/foundry/internal/types"
)

type TelemetryStore struct {
	// Kind of the telemetry store to use.
	Kind TelemetryStoreKind `json:"kind,omitzero" yaml:"kind,omitempty" description:"Kind of the telemetry store to use" examples:"[\"clickhouse\"]"`

	// Specification for the telemetry store.
	Spec MoldingSpec `json:"spec" yaml:"spec" description:"Specification for the telemetry store"`

	// Status of the telemetry store.
	Status TelemetryStoreStatus `json:"status" yaml:"status,omitempty" description:"Status of the telemetry store"`
}

type TelemetryStoreStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	Addresses TelemetryStoreStatusAddresses `json:"addresses" yaml:"addresses,omitempty" description:"Addresses of the telemetry store"`
}

type TelemetryStoreStatusAddresses struct {
	// TCP addresses.
	TCP []string `json:"tcp" yaml:"tcp" description:"TCP addresses"`
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
