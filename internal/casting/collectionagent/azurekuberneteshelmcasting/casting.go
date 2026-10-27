package azurekuberneteshelmcasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	collectionagentcasting "github.com/signoz/foundry/internal/casting/collectionagent"
	"github.com/signoz/foundry/internal/casting/collectionagent/kuberneteshelmcasting"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

var _ collectionagentcasting.Casting = (*azureKubernetesHelmCasting)(nil)

type azureKubernetesHelmCasting struct {
	helmCasting collectionagentcasting.Casting
}

func New(logger *slog.Logger) *azureKubernetesHelmCasting {
	return &azureKubernetesHelmCasting{helmCasting: kuberneteshelmcasting.New(logger)}
}

func (c *azureKubernetesHelmCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	generic, err := c.helmCasting.Enricher(ctx, config)
	if err != nil {
		return nil, err
	}

	return newAzureKubernetesHelmMoldingEnricher(generic), nil
}

func (c *azureKubernetesHelmCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	return c.helmCasting.Forge(ctx, config, p)
}

func (c *azureKubernetesHelmCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	return c.helmCasting.Cast(ctx, config, outputPath, p)
}

func (c *azureKubernetesHelmCasting) Down(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	return c.helmCasting.Down(ctx, config, outputPath, p)
}
