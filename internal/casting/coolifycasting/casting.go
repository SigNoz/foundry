package coolifycasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ rootcasting.Casting = (*coolifyCasting)(nil)

type coolifyCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

func New(logger *slog.Logger) *coolifyCasting {
	return &coolifyCasting{
		logger: logger,
		castings: []*types.Template{
			coolifyYAMLTemplate,
		},
	}
}

func (c *coolifyCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newCoolifyMoldingEnricher(config)
}

func (c *coolifyCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := coolifyYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute coolify yaml template: %w", err)
	}

	coolifyMaterial, err := types.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "coolify.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to create coolify yaml material: %w", err)
	}

	return []types.Material{coolifyMaterial}, nil
}

func (c *coolifyCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Please run 'forge' first to generate the Coolify Casting",
		slog.String("pours_path", poursPath))
	c.logger.InfoContext(ctx, "After forging, deploy coolify.yaml to Coolify using the stack feature",
		slog.String("docs", "https://coolify.io/docs/knowledge-base/docker/compose"))
	return nil
}

func getCoolifyMaterial(config *v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := coolifyYAMLTemplate.Execute(buf, config)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to execute coolify yaml template: %w", err)
	}
	return types.NewYAMLMaterial(buf.Bytes(), path)
}
