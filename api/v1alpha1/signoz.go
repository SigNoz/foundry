package v1alpha1

type SigNoz struct {
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

func NewSigNoz(telemetryStoreDSN string) SigNoz {
	return SigNoz{
		Spec: MoldingSpec{
			Cluster: TypeCluster{
				Replicas: 1,
			},
			Version: "latest",
			Env: map[string]string{
				"SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN": telemetryStoreDSN,
			},
			Config: TypeConfig{
				Data: map[string]string{},
			},
		},
	}
}
