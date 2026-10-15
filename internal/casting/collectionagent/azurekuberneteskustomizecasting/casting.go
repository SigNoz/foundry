package azurekuberneteskustomizecasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	collectionagentcasting "github.com/signoz/foundry/internal/casting/collectionagent"
	"github.com/signoz/foundry/internal/casting/collectionagent/kuberneteskustomizecasting"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

var _ collectionagentcasting.Casting = (*azureKubernetesKustomizeCasting)(nil)

type azureKubernetesKustomizeCasting struct {
	kustomizeCasting collectionagentcasting.Casting
}

func New(logger *slog.Logger) *azureKubernetesKustomizeCasting {
	return &azureKubernetesKustomizeCasting{kustomizeCasting: kuberneteskustomizecasting.New(logger)}
}

func (c *azureKubernetesKustomizeCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	generic, err := c.kustomizeCasting.Enricher(ctx, config)
	if err != nil {
		return nil, err
	}

	return newAzureKubernetesKustomizeMoldingEnricher(generic), nil
}

func (c *azureKubernetesKustomizeCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	return c.kustomizeCasting.Forge(ctx, config, p)
}

func (c *azureKubernetesKustomizeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	return c.kustomizeCasting.Cast(ctx, config, outputPath, p)
}
