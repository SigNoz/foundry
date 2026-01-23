package v1alpha1

type Casting struct {
	TypeVersion `json:",inline" yaml:",inline"`

	// Metadata of the casting configuration.
	Metadata TypeMetadata `json:"metadata" yaml:"metadata"`

	// Specification for the casting.
	Spec CastingSpec `json:"spec" yaml:"spec"`

	// Status of the casting.
	Status CastingStatus `json:"status" yaml:"status"`
}

type CastingSpec struct {
	// Mode platform in which the platform will run.
	Deployment TypeDeployment `json:"deployment" yaml:"deployment"`

	// The configuration for the signoz molding.
	Signoz SigNoz `json:"signoz" yaml:"signoz"`

	// The configuration for the telemetry store molding.
	TelemetryStore TelemetryStore `json:"telemetrystore" yaml:"telemetrystore"`

	// The configuration for the telemetry keeper molding.
	TelemetryKeeper TelemetryKeeper `json:"telemetrykeeper" yaml:"telemetrykeeper"`

	// The configuration for the meta store molding.
	MetaStore MetaStore `json:"metastore" yaml:"metastore"`

	// The configuration for the ingester molding.
	Ingester Ingester `json:"ingester" yaml:"ingester"`
}

type CastingStatus struct {
	// Checksum of the casting file.
	Checksum string `json:"checksum" yaml:"checksum"`
}

func MergeCastingSpecAndStatus(base *Casting) error {
	if err := base.Spec.Signoz.Spec.MergeStatus(base.Spec.Signoz.Status.MoldingStatus); err != nil {
		return err
	}

	if err := base.Spec.TelemetryStore.Spec.MergeStatus(base.Spec.TelemetryStore.Status.MoldingStatus); err != nil {
		return err
	}

	if err := base.Spec.TelemetryKeeper.Spec.MergeStatus(base.Spec.TelemetryKeeper.Status.MoldingStatus); err != nil {
		return err
	}

	if err := base.Spec.MetaStore.Spec.MergeStatus(base.Spec.MetaStore.Status.MoldingStatus); err != nil {
		return err
	}

	if err := base.Spec.Ingester.Spec.MergeStatus(base.Spec.Ingester.Status.MoldingStatus); err != nil {
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
