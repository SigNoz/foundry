package v1alpha1

import (
	"encoding/json"
	"fmt"

	"github.com/signoz/foundry/internal/domain"
)

// SigNozCastingSpec is the spec for kind: SigNoz.
type SigNozCastingSpec struct {
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

// SigNozCastingStatus is the status for kind: SigNoz.
type SigNozCastingStatus struct {
	Checksum string   `json:"checksum" yaml:"checksum" description:"Checksum of the casting file"`
	_        struct{} `additionalProperties:"false"`
}

// kindSigNozCasting is the schema-only variant for kind: SigNoz.
type kindSigNozCasting struct {
	Kind   Kind                `json:"kind" yaml:"kind" required:"true" enum:"SigNoz" description:"Kind discriminator. Must be 'SigNoz'."`
	Spec   SigNozCastingSpec   `json:"spec" yaml:"spec" required:"true" description:"Specification for the SigNoz casting"`
	Status SigNozCastingStatus `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the SigNoz casting"`
}

// SigNozSpec returns the typed spec for kind: SigNoz. Panics if the casting
// is not of kind SigNoz (programmer error — wrong kind at a SigNoz call site).
func (c *Casting) SigNozSpec() *SigNozCastingSpec {
	spec, ok := c.Spec.(*SigNozCastingSpec)
	if !ok {
		panic(fmt.Sprintf("casting kind %q is not SigNoz", c.Kind))
	}
	return spec
}

// SigNozStatus returns the typed status for kind: SigNoz. Panics if the
// casting is not of kind SigNoz.
func (c *Casting) SigNozStatus() *SigNozCastingStatus {
	status, ok := c.Status.(*SigNozCastingStatus)
	if !ok {
		panic(fmt.Sprintf("casting kind %q is not SigNoz", c.Kind))
	}
	return status
}

// DefaultSigNozCasting returns the canonical default for a kind: SigNoz casting.
func DefaultSigNozCasting() Casting {
	return Casting{
		TypeVersion: TypeVersion{APIVersion: "v1alpha1"},
		Kind:        KindSigNoz,
		Metadata:    TypeMetadata{Name: "signoz"},
		Spec: &SigNozCastingSpec{
			Infrastructure:  DefaultInfrastructure(),
			Signoz:          DefaultSigNoz(),
			TelemetryStore:  DefaultTelemetryStore(),
			TelemetryKeeper: DefaultTelemetryKeeper(),
			MetaStore:       DefaultMetaStore(),
			Ingester:        DefaultIngester(),
		},
		Status: &SigNozCastingStatus{},
	}
}

// ExampleSigNozCasting returns a minimal kind: SigNoz casting with only the
// envelope populated; spec is empty so the forge pipeline can fill in defaults.
func ExampleSigNozCasting() Casting {
	return Casting{
		TypeVersion: TypeVersion{APIVersion: "v1alpha1"},
		Kind:        KindSigNoz,
		Metadata:    TypeMetadata{Name: "signoz"},
		Spec:        &SigNozCastingSpec{},
	}
}

// unmarshalSigNozSpecAndStatus decodes raw spec/status payloads into typed
// pointers on c. Called from Casting.UnmarshalJSON when Kind is SigNoz.
func (c *Casting) unmarshalSigNozSpecAndStatus(raw map[string]json.RawMessage) error {
	spec := &SigNozCastingSpec{}
	if rawSpec, ok := raw["spec"]; ok && len(rawSpec) > 0 && string(rawSpec) != "null" {
		if err := json.Unmarshal(rawSpec, spec); err != nil {
			return err
		}
	}
	c.Spec = spec

	status := &SigNozCastingStatus{}
	if rawStatus, ok := raw["status"]; ok && len(rawStatus) > 0 && string(rawStatus) != "null" {
		if err := json.Unmarshal(rawStatus, status); err != nil {
			return err
		}
	}
	c.Status = status
	return nil
}

// signozTrackableProperties extends props with SigNoz-specific analytics fields.
func signozTrackableProperties(props domain.Properties, spec *SigNozCastingSpec) domain.Properties {
	return props.
		Set("platform", spec.Deployment.Platform.String()).
		Set("mode", spec.Deployment.Mode.String()).
		Set("flavor", spec.Deployment.Flavor.String()).
		Set("patches_count", len(spec.Patches)).
		Set("infrastructure_enabled", spec.Infrastructure.Enabled).
		Set("metastore_kind", spec.MetaStore.Kind.String()).
		Set("telemetrystore_kind", spec.TelemetryStore.Kind.String()).
		Set("telemetrykeeper_kind", spec.TelemetryKeeper.Kind.String())
}
