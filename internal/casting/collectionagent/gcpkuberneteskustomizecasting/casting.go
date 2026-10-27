package gcpkuberneteskustomizecasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	collectionagentcasting "github.com/signoz/foundry/internal/casting/collectionagent"
	"github.com/signoz/foundry/internal/casting/collectionagent/kuberneteskustomizecasting"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

var _ collectionagentcasting.Casting = (*gcpKubernetesKustomizeCasting)(nil)

type gcpKubernetesKustomizeCasting struct {
	kustomizeCasting collectionagentcasting.Casting
}

func New(logger *slog.Logger) *gcpKubernetesKustomizeCasting {
	return &gcpKubernetesKustomizeCasting{kustomizeCasting: kuberneteskustomizecasting.New(logger)}
}

func (c *gcpKubernetesKustomizeCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	generic, err := c.kustomizeCasting.Enricher(ctx, config)
	if err != nil {
		return nil, err
	}

	return newGcpKubernetesKustomizeMoldingEnricher(generic), nil
}

func (c *gcpKubernetesKustomizeCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	return c.kustomizeCasting.Forge(ctx, config, p)
}

func (c *gcpKubernetesKustomizeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	return c.kustomizeCasting.Cast(ctx, config, outputPath, p)
}

func (c *gcpKubernetesKustomizeCasting) Down(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	return c.kustomizeCasting.Down(ctx, config, outputPath, p)
}
