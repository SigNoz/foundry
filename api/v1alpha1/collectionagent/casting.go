package collectionagent

import (
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
)

// Casting is the CollectionAgent kind.
type Casting struct {
	v1alpha1.CastingMeta `json:",inline" yaml:",inline"`
	Spec                 Spec     `json:"spec" yaml:"spec" required:"true" description:"CollectionAgent specification"`
	_                    struct{} `additionalProperties:"false"`
}

// Spec is the CollectionAgent-specific configuration.
type Spec struct {
	Deployment v1alpha1.TypeDeployment `json:"deployment" yaml:"deployment" required:"true" description:"Deployment configuration for the platform"`
	Patches    []v1alpha1.PatchEntry   `json:"patches,omitempty" yaml:"patches,omitempty" description:"Patch operations to apply to generated materials"`
	Collector  Collector               `json:"collector,omitzero" yaml:"collector,omitempty" description:"The configuration for the collector molding"`
	_          struct{}                `additionalProperties:"false"`
}

var _ v1alpha1.Machinery = (*Casting)(nil)

// Default returns a CollectionAgent with the collector molding initialised
// from its default.
func Default() *Casting {
	return &Casting{
		CastingMeta: v1alpha1.CastingMeta{
			TypeVersion: v1alpha1.TypeVersion{APIVersion: "v1alpha1"},
			Kind:        v1alpha1.KindCollectionAgent,
			Metadata:    v1alpha1.TypeMetadata{Name: "signoz"},
		},
		Spec: Spec{
			Collector: DefaultCollector(),
		},
	}
}

// Example returns a minimal CollectionAgent; the forge pipeline fills in
// defaults.
func Example() *Casting {
	return &Casting{
		CastingMeta: v1alpha1.CastingMeta{
			TypeVersion: v1alpha1.TypeVersion{APIVersion: "v1alpha1"},
			Kind:        v1alpha1.KindCollectionAgent,
			Metadata:    v1alpha1.TypeMetadata{Name: "signoz"},
		},
	}
}

// Kind reports the casting kind. Shadows the embedded CastingMeta.Kind field;
// the field stays reachable as c.CastingMeta.Kind.
func (c *Casting) Kind() v1alpha1.Kind {
	return v1alpha1.KindCollectionAgent
}

// MergeStatusIntoSpec folds the collector molding's Status into its Spec.
func (c *Casting) MergeStatusIntoSpec() error {
	if err := c.Spec.Collector.Spec.MergeStatus(c.Spec.Collector.Status.MoldingStatus); err != nil {
		return err
	}
	return nil
}

// TrackableProperties returns analytics tags for the casting.
func (c *Casting) TrackableProperties() domain.Properties {
	return domain.NewProperties().
		Set("kind", v1alpha1.KindCollectionAgent.String()).
		Set("platform", c.Spec.Deployment.Platform.String()).
		Set("mode", c.Spec.Deployment.Mode.String()).
		Set("flavor", c.Spec.Deployment.Flavor.String()).
		Set("patches_count", len(c.Spec.Patches)).
		Set("collector_kind", c.Spec.Collector.Kind.String())
}
