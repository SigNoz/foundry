package collectionagent

import "github.com/signoz/foundry/api/v1alpha1"

type Collector struct {
	// Kind of the collector to use.
	Kind CollectorKind `json:"kind,omitzero" yaml:"kind,omitempty" description:"Kind of the collector to use" examples:"[\"agent\"]"`

	// Specification for the collector.
	Spec v1alpha1.MoldingSpec `json:"spec" yaml:"spec" description:"Specification for the collector"`

	// Status of the collector.
	Status CollectorStatus `json:"status" yaml:"status,omitempty" description:"Status of the collector"`

	_ struct{} `additionalProperties:"false"`
}

type CollectorStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	_ struct{} `additionalProperties:"false"`
}

func DefaultCollector() Collector {
	return Collector{
		Kind: CollectorKindAgent,
		Spec: v1alpha1.MoldingSpec{
			Enabled: v1alpha1.BoolPtr(true),
			Cluster: v1alpha1.TypeCluster{
				Replicas: v1alpha1.IntPtr(1),
			},
			Version: "v0.139.0",
			Image:   "otel/opentelemetry-collector-contrib:v0.139.0",
			Env:     map[string]string{},
			Config: v1alpha1.TypeConfig{
				Data: map[string]string{},
			},
		},
	}
}
