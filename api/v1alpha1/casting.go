package v1alpha1

import (
	"encoding/json"
	"fmt"

	"github.com/signoz/foundry/internal/domain"
	"github.com/swaggest/jsonschema-go"
)


type Casting struct {
	TypeVersion `json:",inline" yaml:",inline"`
	Kind        Kind         `json:"kind,omitempty" yaml:"kind,omitempty" description:"Kind of the casting resource. Defaults to SigNoz when omitted."`
	Metadata    TypeMetadata `json:"metadata" yaml:"metadata" required:"true" description:"Metadata of the casting configuration"`
	Spec        any          `json:"spec" yaml:"spec" required:"true" description:"Specification for the casting"`
	Status      any          `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the casting"`
	_           struct{}     `additionalProperties:"false"`
}


type kindSigNozCasting struct {
	Kind   Kind                `json:"kind" yaml:"kind" required:"true" enum:"SigNoz" description:"Kind discriminator. Must be 'SigNoz'."`
	Spec   SigNozCastingSpec   `json:"spec" yaml:"spec" required:"true" description:"Specification for the SigNoz casting"`
	Status SigNozCastingStatus `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the SigNoz casting"`
}

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

func (c *Casting) SigNozSpec() *SigNozCastingSpec {
	spec, ok := c.Spec.(*SigNozCastingSpec)
	if !ok {
		panic(fmt.Sprintf("casting kind %q is not SigNoz", c.Kind))
	}
	return spec
}

func (c *Casting) SigNozStatus() *SigNozCastingStatus {
	status, ok := c.Status.(*SigNozCastingStatus)
	if !ok {
		panic(fmt.Sprintf("casting kind %q is not SigNoz", c.Kind))
	}
	return status
}

var _ jsonschema.OneOfExposer = Casting{}

// JSONSchemaOneOf returns the schema-only variant types per kind.
// Adding a new kind = one entry here plus the concrete spec/status types.
func (Casting) JSONSchemaOneOf() []any {
	return []any{
		kindSigNozCasting{},
	}
}


// UnmarshalJSON decodes envelope fields (apiVersion, kind, metadata) via a
// method-stripped alias, then dispatches Spec and Status into the right
// concrete pointer-typed value based on Kind.
func (c *Casting) UnmarshalJSON(data []byte) error {
	type castingAlias Casting
	if err := json.Unmarshal(data, (*castingAlias)(c)); err != nil {
		return err
	}

	if c.Kind == (Kind{}) {
		c.Kind = KindSigNoz
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	switch c.Kind {
	case KindSigNoz:
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

	default:
		return fmt.Errorf("unknown casting kind %q", c.Kind)
	}

	return nil
}

// DefaultCasting returns the canonical default for a kind: SigNoz casting.
func DefaultCasting() Casting {
	return Casting{
		TypeVersion: TypeVersion{
			APIVersion: "v1alpha1",
		},
		Kind: KindSigNoz,
		Metadata: TypeMetadata{
			Name: "signoz",
		},
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

// ExampleCasting returns a minimal casting with only the deployment spec set.
// The forge pipeline enriches and expands defaults; the full state is written
// to the lock file, not the casting.yaml.
func ExampleCasting() Casting {
	return Casting{
		TypeVersion: TypeVersion{
			APIVersion: "v1alpha1",
		},
		Kind: KindSigNoz,
		Metadata: TypeMetadata{
			Name: "signoz",
		},
		Spec: &SigNozCastingSpec{},
	}
}

// TrackableProperties extracts analytics properties from the casting.
// Dispatches on the concrete Spec type; falls back to envelope-only
// properties when the kind is unknown.
func (c Casting) TrackableProperties() domain.Properties {
	props := domain.NewProperties().Set("kind", c.Kind.String())

	switch spec := c.Spec.(type) {
	case *SigNozCastingSpec:
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
	return props
}
