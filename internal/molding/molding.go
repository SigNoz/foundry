package molding

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
)

type MoldingEnricher interface {
	// Enrich the molding status with the casting configuration.
	EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error
}

type Molding interface {
	// Kind of the molding.
	Kind() v1alpha1.MoldingKind

	// Molds the v1alpha1 casting configuration. This function mutates the config in place. It is not safe for concurrent use.
	MoldV1Alpha1(ctx context.Context, config *installation.Casting) error
}

func MoldingsInOrder() []v1alpha1.MoldingKind {
	return []v1alpha1.MoldingKind{
		v1alpha1.MoldingKindTelemetryKeeper,
		v1alpha1.MoldingKindTelemetryStore,
		v1alpha1.MoldingKindMetaStore,
		v1alpha1.MoldingKindSignoz,
		v1alpha1.MoldingKindIngester,
	}
}
