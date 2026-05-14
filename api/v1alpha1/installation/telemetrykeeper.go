package installation

import "github.com/signoz/foundry/api/v1alpha1"

type TelemetryKeeper struct {
	// Kind of the telemetry keeper to use.
	Kind TelemetryKeeperKind `json:"kind,omitzero" yaml:"kind,omitempty" description:"Kind of the telemetry keeper to use" examples:"[\"clickhousekeeper\"]"`

	// Specification for the telemetry keeper.
	Spec v1alpha1.MoldingSpec `json:"spec" yaml:"spec" description:"Specification for the telemetry keeper"`

	// Status of the telemetry keeper.
	Status TelemetryKeeperStatus `json:"status" yaml:"status,omitempty" description:"Status of the telemetry keeper"`

	_ struct{} `additionalProperties:"false"`
}

type TelemetryKeeperStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	// Addresses of the telemetry keeper.
	Addresses TelemetryKeeperStatusAddresses `json:"addresses" yaml:"addresses,omitempty" description:"Addresses of the telemetry keeper"`

	_ struct{} `additionalProperties:"false"`
}

type TelemetryKeeperStatusAddresses struct {
	// Raft addresses.
	Raft []string `json:"raft" yaml:"raft,omitempty" description:"Raft addresses"`

	// Client addresses.
	Client []string `json:"client" yaml:"client,omitempty" description:"Client addresses"`

	_ struct{} `additionalProperties:"false"`
}

func DefaultTelemetryKeeper() TelemetryKeeper {
	return TelemetryKeeper{
		Kind: TelemetryKeeperKindClickhouseKeeper,
		Spec: v1alpha1.MoldingSpec{
			Enabled: v1alpha1.BoolPtr(true),
			Cluster: v1alpha1.TypeCluster{
				Replicas: v1alpha1.IntPtr(1),
			},
			Version: "25.5.6",
			Image:   "clickhouse/clickhouse-keeper:25.5.6",
		},
	}
}
