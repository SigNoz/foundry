package collectionagent

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/signoz/foundry/internal/runner"
)

type Casting interface {
	Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error)
	Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error
	Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, runners []runner.Runner) error
	Uncast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, runners []runner.Runner) error
}
