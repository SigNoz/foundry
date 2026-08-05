package infrastructure

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
	"github.com/signoz/foundry/internal/pourer"
)

type Casting interface {
	Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error)
	Forge(ctx context.Context, config infrastructure.Casting, p *pourer.Pourer) error
	Cast(ctx context.Context, config infrastructure.Casting, outputPath string, p *pourer.Pourer) error
}
