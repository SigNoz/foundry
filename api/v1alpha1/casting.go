package v1alpha1

type Casting struct {
	TypeVersion `json:",inline" yaml:",inline"`

	// Metadata of the casting configuration.
	Metadata TypeMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Specification for the casting.
	Spec CastingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	// Status of the casting.
	Status CastingStatus `json:"status,omitempty" yaml:"status,omitempty"`
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

type CastingStatus struct {
	// Checksum of the casting file.
	Checksum string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
}

func MergeCastingSpecAndStatus(base *Casting) error {
	if err := base.Spec.Signoz.Spec.MergeStatus(base.Spec.Signoz.Status); err != nil {
		return err
	}

	if err := base.Spec.TelemetryStore.Spec.MergeStatus(base.Spec.TelemetryStore.Status); err != nil {
		return err
	}

	if err := base.Spec.TelemetryKeeper.Spec.MergeStatus(base.Spec.TelemetryKeeper.Status); err != nil {
		return err
	}

	if err := base.Spec.MetaStore.Spec.MergeStatus(base.Spec.MetaStore.Status); err != nil {
		return err
	}

	if err := base.Spec.Ingester.Spec.MergeStatus(base.Spec.Ingester.Status); err != nil {
		return err
	}

	return nil
}

func DefaultCasting() Casting {
	return Casting{
		TypeVersion: TypeVersion{
			APIVersion: "v1alpha1",
		},
		Metadata: TypeMetadata{
			Name: "signoz",
		},
		Spec: CastingSpec{
			Signoz:          DefaultSigNoz(),
			TelemetryStore:  DefaultTelemetryStore(),
			TelemetryKeeper: DefaultTelemetryKeeper(),
			MetaStore:       DefaultMetaStore(),
			Ingester:        DefaultIngester(),
		},
	}
}
