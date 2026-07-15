package ecsec2terraformcasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

type ecsEC2TerraformCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *ecsEC2TerraformCasting {
	return &ecsEC2TerraformCasting{logger: logger}
}

func (c *ecsEC2TerraformCasting) Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error) {
	return &enricher{logger: c.logger}, nil
}

func (c *ecsEC2TerraformCasting) Forge(ctx context.Context, config infrastructure.Casting, poursPath string) ([]domain.Material, error) {
	c.logger.WarnContext(ctx, "the infrastructure kind is not implemented yet, no materials generated")
	return nil, nil
}

func (c *ecsEC2TerraformCasting) Cast(ctx context.Context, config infrastructure.Casting, poursPath string) error {
	c.logger.WarnContext(ctx, "the infrastructure kind is not implemented yet, nothing to cast")
	return nil
}
