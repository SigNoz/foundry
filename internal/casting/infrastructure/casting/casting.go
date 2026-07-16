package casting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

// InfrastructureDir is the subdirectory within the pours directory where
// infrastructure materials are written.
const InfrastructureDir = "infrastructure"

type Casting interface {
	Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error)
	Forge(ctx context.Context, config infrastructure.Casting, poursPath string) ([]domain.Material, error)
	Cast(ctx context.Context, config infrastructure.Casting, poursPath string) error
}
