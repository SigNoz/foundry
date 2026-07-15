package infrastructure

import (
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
)

// Casting is the Infrastructure kind.
type Casting struct {
	v1alpha1.CastingMeta `json:",inline" yaml:",inline"`
	Spec                 Spec     `json:"spec" yaml:"spec" required:"true" description:"Infrastructure specification"`
	_                    struct{} `additionalProperties:"false"`
}

// Spec is the Infrastructure-specific configuration.
type Spec struct {
	Deployment v1alpha1.TypeDeployment  `json:"deployment" yaml:"deployment" required:"true" description:"Deployment configuration for the platform"`
	Resource   v1alpha1.TypeResourceRef `json:"resource" yaml:"resource" required:"true" description:"The resource this infrastructure serves"`
	Patches    []v1alpha1.PatchEntry    `json:"patches,omitempty" yaml:"patches,omitempty" description:"Patch operations to apply to generated materials"`
	_          struct{}                 `additionalProperties:"false"`
}

var _ v1alpha1.Machinery = (*Casting)(nil)

// Default returns an Infrastructure casting with defaults initialised.
func Default() *Casting {
	return &Casting{
		CastingMeta: v1alpha1.CastingMeta{
			TypeVersion: v1alpha1.TypeVersion{APIVersion: "v1alpha1"},
			Kind:        v1alpha1.KindInfrastructure,
			Metadata:    v1alpha1.TypeMetadata{Name: "signoz"},
		},
		Spec: Spec{
			Deployment: v1alpha1.TypeDeployment{Flavor: v1alpha1.FlavorTerraform},
		},
	}
}

// Example returns a minimal Infrastructure; the forge pipeline fills in
// defaults.
func Example() *Casting {
	return &Casting{
		CastingMeta: v1alpha1.CastingMeta{
			TypeVersion: v1alpha1.TypeVersion{APIVersion: "v1alpha1"},
			Kind:        v1alpha1.KindInfrastructure,
			Metadata:    v1alpha1.TypeMetadata{Name: "signoz"},
		},
	}
}

// Kind reports the casting kind. Shadows the embedded CastingMeta.Kind field;
// the field stays reachable as c.CastingMeta.Kind.
func (c *Casting) Kind() v1alpha1.Kind {
	return v1alpha1.KindInfrastructure
}

// MergeStatusIntoSpec folds molding-written status into spec. The resource
// reference's status is its own home; nothing shadows spec fields.
func (c *Casting) MergeStatusIntoSpec() error {
	return nil
}

// TrackableProperties returns analytics tags for the casting.
func (c *Casting) TrackableProperties() domain.Properties {
	return domain.NewProperties().
		Set("kind", v1alpha1.KindInfrastructure.String()).
		Set("platform", c.Spec.Deployment.Platform.String()).
		Set("flavor", c.Spec.Deployment.Flavor.String()).
		Set("resource_kind", c.Spec.Resource.Kind.String()).
		Set("patches_count", len(c.Spec.Patches))
}
