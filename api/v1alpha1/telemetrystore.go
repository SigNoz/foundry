package v1alpha1

type TelemetryStore struct {
	// Kind of the telemetry store to use.
	Kind TelemetryStoreKind `json:"kind,omitzero" yaml:"kind,omitempty" description:"Kind of the telemetry store to use" examples:"[\"clickhouse\"]"`

	// Specification for the telemetry store.
	Spec MoldingSpec `json:"spec" yaml:"spec" description:"Specification for the telemetry store"`

	// Status of the telemetry store.
	Status TelemetryStoreStatus `json:"status" yaml:"status,omitempty" description:"Status of the telemetry store"`

	_ struct{} `additionalProperties:"false"`
}

type TelemetryStoreStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

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
		Spec: MoldingSpec{
			Enabled: boolPtr(true),
			Cluster: TypeCluster{
				Replicas: intPtr(0),
				Shards:   intPtr(1),
			},
			Version: "25.5.6",
			Image:   "clickhouse/clickhouse-server:25.5.6",
		},
	}
}
