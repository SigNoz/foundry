package azurekuberneteskustomizecasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/casting/collectionagent/kuberneteskustomizecasting"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

// The registry's package imports this one, so its Casting interface is restated here.
type kustomizeCasting interface {
	Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error)
	Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error
	Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error
}

type azureKubernetesKustomizeCasting struct {
	kustomizeCasting kustomizeCasting
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
