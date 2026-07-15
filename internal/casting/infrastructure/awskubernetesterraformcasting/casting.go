package awskubernetesterraformcasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

type awsKubernetesTerraformCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *awsKubernetesTerraformCasting {
	return &awsKubernetesTerraformCasting{logger: logger}
}

func (c *awsKubernetesTerraformCasting) Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error) {
	return &enricher{logger: c.logger}, nil
}

func (c *awsKubernetesTerraformCasting) Forge(ctx context.Context, config infrastructure.Casting, poursPath string) ([]domain.Material, error) {
	c.logger.WarnContext(ctx, "the infrastructure kind is not implemented yet, no materials generated")
	return nil, nil
}

func (c *awsKubernetesTerraformCasting) Cast(ctx context.Context, config infrastructure.Casting, poursPath string) error {
	c.logger.WarnContext(ctx, "the infrastructure kind is not implemented yet, nothing to cast")
	return nil
}
