package coolifycasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
)

var _ rootcasting.Casting = (*coolifyCasting)(nil)

type coolifyCasting struct {
	logger   *slog.Logger
	castings []*domain.Template
}

func New(logger *slog.Logger) *coolifyCasting {
	return &coolifyCasting{
		logger: logger,
		castings: []*domain.Template{
			coolifyYAMLTemplate,
		},
	}
}

func (c *coolifyCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newCoolifyMoldingEnricher(config)
}

func (c *coolifyCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := coolifyYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute coolify yaml template: %w", err)
	}

	coolifyMaterial, err := domain.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "coolify.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to create coolify yaml material: %w", err)
	}

	return []domain.Material{coolifyMaterial}, nil
}

func (c *coolifyCasting) Cast(ctx context.Context, config installation.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Please run 'forge' first to generate the Coolify Casting",
		slog.String("pours_path", poursPath))
	c.logger.InfoContext(ctx, "After forging, deploy coolify.yaml to Coolify using the stack feature",
		slog.String("docs", "https://coolify.io/docs/knowledge-base/docker/compose"))
	return nil
}

func getCoolifyMaterial(config *installation.Casting, path string) (domain.StructuredMaterial, error) {
	buf := bytes.NewBuffer(nil)
	err := coolifyYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute coolify yaml template: %w", err)
	}
	return domain.NewYAMLMaterial(buf.Bytes(), path)
}
