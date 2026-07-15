package terraformcasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

type terraformCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *terraformCasting {
	return &terraformCasting{logger: logger}
}

func (c *terraformCasting) Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error) {
	return &enricher{logger: c.logger}, nil
}

func (c *terraformCasting) Forge(ctx context.Context, config infrastructure.Casting, poursPath string) ([]domain.Material, error) {
	c.logger.InfoContext(ctx, "infrastructure terraform casting is scaffolding, no materials generated yet")
	return nil, nil
}

func (c *terraformCasting) Cast(ctx context.Context, config infrastructure.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "infrastructure terraform casting is scaffolding, nothing to cast yet")
	return nil
}
