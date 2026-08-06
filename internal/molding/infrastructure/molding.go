package infrastructure

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/convention"
)

type MoldingEnricher interface {
	EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *infrastructure.Casting) error
}

type Molding interface {
	Kind() v1alpha1.MoldingKind
	MoldV1Alpha1(ctx context.Context, config *infrastructure.Casting) error
}

// Deriver turns a settled declaration into every name and tag the substrate
// stamps. The topology is the provider's. The registry supplies one per
// deployment.
type Deriver func(substrate convention.Substrate, declaration *infrastructure.ResourceConfig, labels map[string]string) (*infrastructure.ResourceConfigResources, error)
