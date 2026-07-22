package collectionagent

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

type Casting interface {
	Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error)
	Forge(ctx context.Context, config collectionagent.Casting, poursPath string) ([]domain.Material, error)
	Cast(ctx context.Context, config collectionagent.Casting, poursPath string) error
}
