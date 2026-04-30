package v1alpha1

type Casting struct {
	TypeVersion `json:",inline" yaml:",inline"`
	Metadata    TypeMetadata  `json:"metadata" yaml:"metadata" required:"true" description:"Metadata of the casting configuration"`
	Spec        CastingSpec   `json:"spec" yaml:"spec" required:"true" description:"Specification for the casting"`
	Status      CastingStatus `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the casting"`
	_           struct{}      `additionalProperties:"false"`
}

type CastingSpec struct {
	Deployment      TypeDeployment  `json:"deployment" yaml:"deployment" required:"true" description:"Deployment configuration for the platform"`
	Infrastructure  Infrastructure  `json:"infrastructure,omitzero" yaml:"infrastructure,omitzero" description:"Infrastructure configuration for generating infrastructure manifests (e.g., Terraform)."`
	Signoz          SigNoz          `json:"signoz,omitzero" yaml:"signoz,omitempty" description:"The configuration for the SigNoz molding"`
	TelemetryStore  TelemetryStore  `json:"telemetrystore,omitzero" yaml:"telemetrystore,omitempty" description:"The configuration for the telemetry store molding"`
	TelemetryKeeper TelemetryKeeper `json:"telemetrykeeper,omitzero" yaml:"telemetrykeeper,omitempty" description:"The configuration for the telemetry keeper molding"`
	MetaStore       MetaStore       `json:"metastore,omitzero" yaml:"metastore,omitempty" description:"The configuration for the meta store molding"`
	Ingester        Ingester        `json:"ingester,omitzero" yaml:"ingester,omitempty" description:"The configuration for the ingester molding"`
	Patches         []PatchEntry    `json:"patches,omitempty" yaml:"patches,omitempty" description:"Patch operations to apply to generated materials"`
	_               struct{}        `additionalProperties:"false"`
}

type CastingStatus struct {
	// Checksum of the casting file.
	Checksum string   `json:"checksum" yaml:"checksum" description:"Checksum of the casting file"`
	_        struct{} `additionalProperties:"false"`
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
			Infrastructure:  DefaultInfrastructure(),
			Signoz:          DefaultSigNoz(),
			TelemetryStore:  DefaultTelemetryStore(),
			TelemetryKeeper: DefaultTelemetryKeeper(),
			MetaStore:       DefaultMetaStore(),
			Ingester:        DefaultIngester(),
		},
	}
}

// ExampleCasting returns a minimal casting with only the deployment spec set.
// The forge pipeline enriches and expands defaults; the full state is written
// to the lock file, not the casting.yaml.
func ExampleCasting() Casting {
	return Casting{
		TypeVersion: TypeVersion{
			APIVersion: "v1alpha1",
		},
		Metadata: TypeMetadata{
			Name: "signoz",
		},
		Spec: CastingSpec{},
	}
}
