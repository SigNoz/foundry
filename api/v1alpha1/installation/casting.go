package installation

import (
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
)

// Casting is the Installation kind.
type Casting struct {
	v1alpha1.CastingMeta `json:",inline" yaml:",inline"`
	Spec                 Spec     `json:"spec" yaml:"spec" required:"true" description:"Installation specification"`
	_                    struct{} `additionalProperties:"false"`
}

// Spec is the Installation-specific configuration.
type Spec struct {
	Deployment      v1alpha1.TypeDeployment `json:"deployment" yaml:"deployment" required:"true" description:"Deployment configuration for the platform"`
	Patches         []v1alpha1.PatchEntry   `json:"patches,omitempty" yaml:"patches,omitempty" description:"Patch operations to apply to generated materials"`
	Infrastructure  Infrastructure          `json:"infrastructure,omitzero" yaml:"infrastructure,omitzero" description:"Infrastructure configuration for generating infrastructure manifests (e.g., Terraform)."`
	Signoz          SigNoz                  `json:"signoz,omitzero" yaml:"signoz,omitempty" description:"The configuration for the SigNoz molding"`
	TelemetryStore  TelemetryStore          `json:"telemetrystore,omitzero" yaml:"telemetrystore,omitempty" description:"The configuration for the telemetry store molding"`
	TelemetryKeeper TelemetryKeeper         `json:"telemetrykeeper,omitzero" yaml:"telemetrykeeper,omitempty" description:"The configuration for the telemetry keeper molding"`
	MetaStore       MetaStore               `json:"metastore,omitzero" yaml:"metastore,omitempty" description:"The configuration for the meta store molding"`
	Ingester        Ingester                `json:"ingester,omitzero" yaml:"ingester,omitempty" description:"The configuration for the ingester molding"`
	_               struct{}                `additionalProperties:"false"`
}

var _ v1alpha1.Machinery = (*Casting)(nil)

// Default returns an Installation with every molding initialised from its
// default.
func Default() *Casting {
	return &Casting{
		CastingMeta: v1alpha1.CastingMeta{
			TypeVersion: v1alpha1.TypeVersion{APIVersion: "v1alpha1"},
			Kind:        v1alpha1.KindInstallation,
			Metadata:    v1alpha1.TypeMetadata{Name: "signoz"},
		},
		Spec: Spec{
			Infrastructure:  DefaultInfrastructure(),
			Signoz:          DefaultSigNoz(),
			TelemetryStore:  DefaultTelemetryStore(),
			TelemetryKeeper: DefaultTelemetryKeeper(),
			MetaStore:       DefaultMetaStore(),
			Ingester:        DefaultIngester(),
		},
	}
}

// Example returns a minimal Installation; the forge pipeline fills in defaults.
func Example() *Casting {
	return &Casting{
		CastingMeta: v1alpha1.CastingMeta{
			TypeVersion: v1alpha1.TypeVersion{APIVersion: "v1alpha1"},
			Kind:        v1alpha1.KindInstallation,
			Metadata:    v1alpha1.TypeMetadata{Name: "signoz"},
		},
	}
}

// Kind reports the casting kind. Shadows the embedded CastingMeta.Kind field;
// the field stays reachable as c.CastingMeta.Kind.
func (c *Casting) Kind() v1alpha1.Kind {
	return v1alpha1.KindInstallation
}

// MergeStatusIntoSpec folds each molding's Status into its own Spec.
func (c *Casting) MergeStatusIntoSpec() error {
	if err := c.Spec.Signoz.Spec.MergeStatus(c.Spec.Signoz.Status.MoldingStatus); err != nil {
		return err
	}
	if err := c.Spec.TelemetryStore.Spec.MergeStatus(c.Spec.TelemetryStore.Status.MoldingStatus); err != nil {
		return err
	}
	if err := c.Spec.TelemetryKeeper.Spec.MergeStatus(c.Spec.TelemetryKeeper.Status.MoldingStatus); err != nil {
		return err
	}
	if err := c.Spec.MetaStore.Spec.MergeStatus(c.Spec.MetaStore.Status.MoldingStatus); err != nil {
		return err
	}
	if err := c.Spec.Ingester.Spec.MergeStatus(c.Spec.Ingester.Status.MoldingStatus); err != nil {
		return err
	}
	return nil
}

// TrackableProperties returns analytics tags for the casting.
func (c *Casting) TrackableProperties() domain.Properties {
	return domain.NewProperties().
		Set("kind", v1alpha1.KindInstallation.String()).
		Set("platform", c.Spec.Deployment.Platform.String()).
		Set("mode", c.Spec.Deployment.Mode.String()).
		Set("flavor", c.Spec.Deployment.Flavor.String()).
		Set("patches_count", len(c.Spec.Patches)).
		Set("infrastructure_enabled", c.Spec.Infrastructure.Enabled).
		Set("metastore_kind", c.Spec.MetaStore.Kind.String()).
		Set("telemetrystore_kind", c.Spec.TelemetryStore.Kind.String()).
		Set("telemetrykeeper_kind", c.Spec.TelemetryKeeper.Kind.String())
}
