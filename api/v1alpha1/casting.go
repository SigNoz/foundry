package v1alpha1

type Casting struct {
	TypeVersion `json:",inline" yaml:",inline"`

	// Metadata of the casting configuration.
	Metadata TypeMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Specification for the casting.
	Spec CastingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

type CastingSpec struct {
	// Mode platform in which the platform will run.
	Deployment TypeDeployment `json:"deployment,omitempty" yaml:"deployment,omitempty"`

	// The configuration for the signoz molding.
	Signoz SigNoz `json:"signoz,omitempty" yaml:"signoz,omitempty"`

	// The configuration for the telemetry store molding.
	TelemetryStore TelemetryStore `json:"telemetrystore,omitempty" yaml:"telemetrystore,omitempty"`

	// The configuration for the telemetry keeper molding.
	TelemetryKeeper TelemetryKeeper `json:"telemetrykeeper,omitempty" yaml:"telemetrykeeper,omitempty"`

	// The configuration for the meta store molding.
	MetaStore MetaStore `json:"metastore,omitempty" yaml:"metastore,omitempty"`

	// The configuration for the ingester molding.
	Ingester Ingester `json:"ingester,omitempty" yaml:"ingester,omitempty"`
}

func (c Casting) MergeStatusIntoSpec() Casting {
	return Casting{
		TypeVersion: c.TypeVersion,
		Metadata:    c.Metadata,
		Spec:        c.Spec.MergeStatusIntoSpec(),
	}
}

func (s CastingSpec) MergeStatusIntoSpec() CastingSpec {
	return CastingSpec{
		Deployment:      s.Deployment,
		Signoz:          s.Signoz.MergeStatusIntoSpec(),
		TelemetryStore:  s.TelemetryStore.MergeStatusIntoSpec(),
		TelemetryKeeper: s.TelemetryKeeper.MergeStatusIntoSpec(),
		MetaStore:       s.MetaStore.MergeStatusIntoSpec(),
		Ingester:        s.Ingester.MergeStatusIntoSpec(),
	}
}
