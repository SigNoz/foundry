package v1alpha1

import "github.com/signoz/foundry/api/v1alpha1/yamls"

type Ingester struct {
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

func NewIngester(telemetryStoreDSN string) Ingester {
	return Ingester{
		Spec: MoldingSpec{
			Cluster: TypeCluster{
				Replicas: 1,
			},
			Version: "latest",
			Env: map[string]string{
				"SIGNOZOTELCOLLECTOR_TELEMETRYSTORE_DSN": telemetryStoreDSN,
				"LOW_CARDINAL_EXCEPTION_GROUPING":        "true",
			},
			Config: TypeConfig{
				Data: map[string]string{
					"config.yaml": yamls.ConfigIngesterV0129xYAML,
				},
			},
		},
	}
}
