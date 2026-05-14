package installation

import "github.com/signoz/foundry/api/v1alpha1"

type TelemetryStore struct {
	// Kind of the telemetry store to use.
	Kind TelemetryStoreKind `json:"kind,omitzero" yaml:"kind,omitempty" description:"Kind of the telemetry store to use" examples:"[\"clickhouse\"]"`

	// Specification for the telemetry store.
	Spec v1alpha1.MoldingSpec `json:"spec" yaml:"spec" description:"Specification for the telemetry store"`

	// Status of the telemetry store.
	Status TelemetryStoreStatus `json:"status" yaml:"status,omitempty" description:"Status of the telemetry store"`

	_ struct{} `additionalProperties:"false"`
}

type TelemetryStoreStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	Addresses TelemetryStoreStatusAddresses `json:"addresses" yaml:"addresses,omitempty" description:"Addresses of the telemetry store"`

	_ struct{} `additionalProperties:"false"`
}

type TelemetryStoreStatusAddresses struct {
	// TCP addresses.
	TCP []string `json:"tcp" yaml:"tcp" description:"TCP addresses"`

	_ struct{} `additionalProperties:"false"`
}

func DefaultTelemetryStore() TelemetryStore {
	return TelemetryStore{
		Kind: TelemetryStoreKindClickhouse,
		Spec: v1alpha1.MoldingSpec{
			Enabled: v1alpha1.BoolPtr(true),
			Cluster: v1alpha1.TypeCluster{
				Replicas: v1alpha1.IntPtr(0),
				Shards:   v1alpha1.IntPtr(1),
			},
			Version: "25.5.6",
			Image:   "clickhouse/clickhouse-server:25.5.6",
		},
	}
}
