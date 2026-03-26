package v1alpha1

type Casting struct {
	TypeVersion `json:",inline" yaml:",inline"`

	// Metadata of the casting configuration.
	Metadata TypeMetadata `json:"metadata" yaml:"metadata" description:"Metadata of the casting configuration"`

	// Specification for the casting.
	Spec CastingSpec `json:"spec" yaml:"spec" description:"Specification for the casting"`

	// Status of the casting.
	Status CastingStatus `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the casting"`
}

type CastingSpec struct {
	// Mode platform in which the platform will run.
	Deployment TypeDeployment `json:"deployment" yaml:"deployment" description:"Deployment configuration for the platform"`

	// Infrastructure configuration for generating infrastructure manifests (e.g., Terraform).
	Infrastructure Infrastructure `json:"infrastructure,omitzero" yaml:"infrastructure,omitzero"`
	// Patches are patch operations applied to generated output files.
	Patches []PatchEntry `json:"patches,omitempty" yaml:"patches,omitempty" description:"Patch operations to apply to generated materials"`

	// The configuration for the signoz molding.
	Signoz SigNoz `json:"signoz,omitzero" yaml:"signoz,omitempty" description:"The configuration for the SigNoz molding"`

	// The configuration for the telemetry store molding.
	TelemetryStore TelemetryStore `json:"telemetrystore,omitzero" yaml:"telemetrystore,omitempty" description:"The configuration for the telemetry store molding"`

	// The configuration for the telemetry keeper molding.
	TelemetryKeeper TelemetryKeeper `json:"telemetrykeeper,omitzero" yaml:"telemetrykeeper,omitempty" description:"The configuration for the telemetry keeper molding"`

	// The configuration for the meta store molding.
	MetaStore MetaStore `json:"metastore,omitzero" yaml:"metastore,omitempty" description:"The configuration for the meta store molding"`

	// The configuration for the ingester molding.
	Ingester Ingester `json:"ingester,omitzero" yaml:"ingester,omitempty" description:"The configuration for the ingester molding"`
}

type CastingStatus struct {
	// Checksum of the casting file.
	Checksum string `json:"checksum" yaml:"checksum" description:"Checksum of the casting file"`
}

const (
	// PatchTypeJSONPatch is the default patch type using JSON Patch (RFC 6902).
	PatchTypeJSONPatch = "jsonpatch"
)

// PatchEntry is a set of patch operations targeting a specific generated file.
type PatchEntry struct {
	// Type selects the patch driver. Defaults to "jsonpatch" if empty.
	Type string `json:"type,omitempty" yaml:"type,omitempty" description:"Patch driver type. Defaults to jsonpatch." default:"jsonpatch" example:"jsonpatch"`

	// Target is the output file to patch, relative to the pours directory.
	Target string `json:"target,omitempty" yaml:"target,omitempty" description:"Target output file to patch" examples:"[\"compose.yaml\",\"signoz/deployment.yaml\",\"values.yaml\",\"telemetrystore/telemtrystore-clickhouse-0-*.yaml\"]"`

	// Operations is a list of JSON Patch (RFC 6902) operations to apply. Used by the jsonpatch driver.
	Operations []PatchOperation `json:"operations,omitempty" yaml:"operations,omitempty" description:"JSON Patch (RFC 6902) operations to apply. Used by the jsonpatch driver."`
}

// PatchType returns the patch type, defaulting to PatchTypeJSONPatch if empty.
func (pe PatchEntry) PatchType() string {
	if pe.Type == "" {
		return PatchTypeJSONPatch
	}
	return pe.Type
}

// PatchOperation is a single JSON Patch (RFC 6902) operation. Used by the jsonpatch driver.
type PatchOperation struct {
	// Op is the JSON Patch (RFC 6902) operation type: add, remove, replace, move, copy, test.
	Op string `json:"op" yaml:"op" description:"JSON Patch (RFC 6902) operation type" examples:"[\"add\",\"remove\",\"replace\",\"move\",\"copy\",\"test\"]"`

	// Path is a JSON Pointer (RFC 6902) to the target location.
	Path string `json:"path" yaml:"path" description:"JSON Pointer (RFC 6901) to the target location" example:"/services/clickhouse/mem_limit"`

	// Value is the value for add, replace, or test operations.
	Value any `json:"value,omitempty" yaml:"value,omitempty" description:"Value for add, replace, or test operations"`

	// From is a JSON Pointer for the source location in move and copy operations.
	From string `json:"from,omitempty" yaml:"from,omitempty" description:"Source JSON Pointer for move and copy operations" example:"/services/clickhouse/old_field"`
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
