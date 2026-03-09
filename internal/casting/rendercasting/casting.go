package rendercasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ casting.Casting = (*renderCasting)(nil)

type renderCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

func New(logger *slog.Logger) *renderCasting {
	return &renderCasting{
		logger: logger,
		castings: []*types.Template{
			renderYAMLTemplate,
			telemetryKeeperDockerfileTemplate,
			telemetryStoreDockerfileTemplate,
			ingesterDockerfileTemplate,
		},
	}
}

func (c *renderCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newRenderMoldingEnricher(config)
}

func (c *renderCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	var materials []types.Material

	configsDir := filepath.Join(casting.DeploymentDir, "configs/")
	// Generate render.yaml
	blueprintMaterial, err := getRenderMaterial(&config, filepath.Join(casting.DeploymentDir, "render.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to create blueprint yaml material: %w", err)
	}
	materials = append(materials, blueprintMaterial)

	// Generate Dockerfile for telemetrykeeper services
	if config.Spec.TelemetryKeeper.Spec.Enabled {
		dockerfileBuf := bytes.NewBuffer(nil)
		err := telemetryKeeperDockerfileTemplate.Execute(dockerfileBuf, config)
		if err != nil {
			return nil, fmt.Errorf("failed to execute dockerfile keeper template: %w", err)
		}
		dockerfileMaterial := types.NewTextMaterial(dockerfileBuf.Bytes(), filepath.Join(configsDir, "telemetrykeeper/Dockerfile"))
		materials = append(materials, dockerfileMaterial)

		// Add telemetrykeeper config files (for dockerfile to copy)
		for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
			material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(configsDir, fmt.Sprintf("telemetrykeeper/keeper.d/%s", filename)))
			if err != nil {
				return nil, fmt.Errorf("failed to create telemetrykeeper config material: %w", err)
			}
			materials = append(materials, material)
		}
	}

	// Add Dockerfile for telemetrystore services
	if config.Spec.TelemetryStore.Spec.Enabled {
		dockerfileBuf := bytes.NewBuffer(nil)
		err := telemetryStoreDockerfileTemplate.Execute(dockerfileBuf, config)
		if err != nil {
			return nil, fmt.Errorf("failed to execute dockerfile clickhouse template: %w", err)
		}
		dockerfileMaterial := types.NewTextMaterial(dockerfileBuf.Bytes(), filepath.Join(configsDir, "telemetrystore/Dockerfile"))
		materials = append(materials, dockerfileMaterial)

		// Add telemetrystore config files (for dockerfile to copy)
		for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
			material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(configsDir, fmt.Sprintf("telemetrystore/config.d/%s", filename)))
			if err != nil {
				return nil, fmt.Errorf("failed to create telemetrystore config material: %w", err)
			}
			materials = append(materials, material)
		}
	}

	// Add Dockerfile for ingester services
	if config.Spec.Ingester.Spec.Enabled {
		dockerfileBuf := bytes.NewBuffer(nil)
		err := ingesterDockerfileTemplate.Execute(dockerfileBuf, config)
		if err != nil {
			return nil, fmt.Errorf("failed to execute dockerfile otel template: %w", err)
		}
		dockerfileMaterial := types.NewTextMaterial(dockerfileBuf.Bytes(), filepath.Join(configsDir, "ingester/Dockerfile"))
		materials = append(materials, dockerfileMaterial)

		for filename, content := range config.Spec.Ingester.Spec.Config.Data {
			material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(configsDir, fmt.Sprintf("ingester/%s", filename)))
			if err != nil {
				return nil, fmt.Errorf("failed to create ingester config material: %w", err)
			}
			materials = append(materials, material)
		}
	}

	return materials, nil
}

func (c *renderCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Please run 'forge' first to generate the Render Casting",
		slog.String("pours_path", poursPath))
	c.logger.InfoContext(ctx, "After forging, deploy render.yaml to Render using Infrastructure as Code",
		slog.String("Docs", "https://render.com/docs/infrastructure-as-code#setup"))
	return nil
}

func (c *renderCasting) NeedsMoldings() bool {
	return true
}

func getRenderMaterial(config *v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := renderYAMLTemplate.Execute(buf, config)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to execute render yaml template: %w", err)
	}
	return types.NewYAMLMaterial(buf.Bytes(), path)
}
