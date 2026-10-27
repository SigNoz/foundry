package gcpkuberneteshelmcasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	collectionagentcasting "github.com/signoz/foundry/internal/casting/collectionagent"
	"github.com/signoz/foundry/internal/casting/collectionagent/kuberneteshelmcasting"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

var _ collectionagentcasting.Casting = (*gcpKubernetesHelmCasting)(nil)

type gcpKubernetesHelmCasting struct {
	helmCasting collectionagentcasting.Casting
}

func New(logger *slog.Logger) *gcpKubernetesHelmCasting {
	return &gcpKubernetesHelmCasting{helmCasting: kuberneteshelmcasting.New(logger)}
}

func (c *gcpKubernetesHelmCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	generic, err := c.helmCasting.Enricher(ctx, config)
	if err != nil {
		return nil, err
	}

	return newGcpKubernetesHelmMoldingEnricher(generic), nil
}

func (c *gcpKubernetesHelmCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	return c.helmCasting.Forge(ctx, config, p)
}

func (c *gcpKubernetesHelmCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	return c.helmCasting.Cast(ctx, config, outputPath, p)
}

func (c *gcpKubernetesHelmCasting) Down(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	return c.helmCasting.Down(ctx, config, outputPath, p)
}
