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

// CollectorStatus is the contract surface between casting enrichers (which
// fill the slots) and the collector molding (which renders the mold cavity).
// Fields carry platform-necessary OTel components the casting contributes;
// each Component map entry colocates its body with the pipelines it plugs into.
type CollectorStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	Receivers         map[string]Component      `json:"receivers,omitempty" yaml:"receivers,omitempty" description:"OTel receivers contributed by the casting, with the pipelines each feeds"`
	Processors        map[string]Component      `json:"processors,omitempty" yaml:"processors,omitempty" description:"OTel processors contributed by the casting, with the pipelines each runs in"`
	Exporters         map[string]Component      `json:"exporters,omitempty" yaml:"exporters,omitempty" description:"OTel exporters contributed by the casting, with the pipelines each receives from"`
	Extensions        map[string]map[string]any `json:"extensions,omitempty" yaml:"extensions,omitempty" description:"OTel extensions contributed by the casting; declared entries are activated under service.extensions"`
	ResourceDetectors []string                  `json:"resourceDetectors,omitempty" yaml:"resourceDetectors,omitempty" description:"Detectors appended to the baked-in [env, system] list on the resourcedetection processor"`

	_ struct{} `additionalProperties:"false"`
}

// Component pairs an OTel component body with the pipelines it participates in.
type Component struct {
	Body      map[string]any `json:"body" yaml:"body" description:"OTel component body, an opaque map preserving the upstream schema"`
	Pipelines []string       `json:"pipelines,omitempty" yaml:"pipelines,omitempty" description:"Pipelines (traces|metrics|logs) this component plugs into"`
	_         struct{}       `additionalProperties:"false"`
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
