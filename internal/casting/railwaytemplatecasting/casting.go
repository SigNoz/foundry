package railwaytemplatecasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
)

var _ casting.Casting = (*railwayTemplateCasting)(nil)

type railwayTemplateCasting struct {
	logger   *slog.Logger
	castings []*domain.Template
}

func New(logger *slog.Logger) *railwayTemplateCasting {
	return &railwayTemplateCasting{
		logger: logger,
		castings: []*domain.Template{
			telemetryKeeperDockerfileTemplate,
			telemetryStoreDockerfileTemplate,
			ingesterDockerfileTemplate,
			signozDockerfileTemplate,
			telemetryStoreMigratorDockerfileTemplate,
			railwayTelemetryKeeperTemplate,
			railwayTelemetryStoreTemplate,
			railwayIngesterTemplate,
			railwaySignozTemplate,
			railwayTelemetryStoreMigratorTemplate,
			telemetryKeeperOverrideTemplate,
			telemetryStoreOverrideTemplate,
		},
	}
}

func (c *railwayTemplateCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newRailwayTemplateMoldingEnricher(config)
}

func (c *railwayTemplateCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {
	var materials []domain.Material

	// TelemetryKeeper: Dockerfile + configs + railway.json
	if config.Spec.TelemetryKeeper.Spec.IsEnabled() {
		dockerfileBuf := bytes.NewBuffer(nil)
		if err := telemetryKeeperDockerfileTemplate.Execute(dockerfileBuf, config); err != nil {
			return nil, fmt.Errorf("telemetrykeeper dockerfile: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(dockerfileBuf.Bytes(), filepath.Join(casting.DeploymentDir, "telemetrykeeper/Dockerfile")))
		railwayBuf := bytes.NewBuffer(nil)
		if err := railwayTelemetryKeeperTemplate.Execute(railwayBuf, config); err != nil {
			return nil, fmt.Errorf("telemetrykeeper railway.json: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(railwayBuf.Bytes(), filepath.Join(casting.DeploymentDir, "telemetrykeeper/railway.json")))
		for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
			m, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(casting.DeploymentDir, "telemetrykeeper/keeper.d/", filename))
			if err != nil {
				return nil, fmt.Errorf("telemetrykeeper config: %w", err)
			}
			materials = append(materials, m)
		}
	}

	// TelemetryStore: Dockerfile + configs + railway.json
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		dockerfileBuf := bytes.NewBuffer(nil)
		if err := telemetryStoreDockerfileTemplate.Execute(dockerfileBuf, config); err != nil {
			return nil, fmt.Errorf("telemetrystore dockerfile: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(dockerfileBuf.Bytes(), filepath.Join(casting.DeploymentDir, "telemetrystore/Dockerfile")))
		railwayBuf := bytes.NewBuffer(nil)
		if err := railwayTelemetryStoreTemplate.Execute(railwayBuf, config); err != nil {
			return nil, fmt.Errorf("telemetrystore railway.json: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(railwayBuf.Bytes(), filepath.Join(casting.DeploymentDir, "telemetrystore/railway.json")))
		for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
			m, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(casting.DeploymentDir, "telemetrystore/config.d/", filename))
			if err != nil {
				return nil, fmt.Errorf("telemetrystore config: %w", err)
			}
			materials = append(materials, m)
		}
	}

	// Ingester: Dockerfile + configs + railway.json
	if config.Spec.Ingester.Spec.IsEnabled() {
		dockerfileBuf := bytes.NewBuffer(nil)
		if err := ingesterDockerfileTemplate.Execute(dockerfileBuf, config); err != nil {
			return nil, fmt.Errorf("ingester dockerfile: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(dockerfileBuf.Bytes(), filepath.Join(casting.DeploymentDir, "ingester/Dockerfile")))
		railwayBuf := bytes.NewBuffer(nil)
		if err := railwayIngesterTemplate.Execute(railwayBuf, config); err != nil {
			return nil, fmt.Errorf("ingester railway.json: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(railwayBuf.Bytes(), filepath.Join(casting.DeploymentDir, "ingester/railway.json")))
		for filename, content := range config.Spec.Ingester.Spec.Config.Data {
			m, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(casting.DeploymentDir, "ingester/", filename))
			if err != nil {
				return nil, fmt.Errorf("ingester config: %w", err)
			}
			materials = append(materials, m)
		}
	}

	// Signoz: Dockerfile + railway.json
	if config.Spec.Signoz.Spec.IsEnabled() {
		dockerfileBuf := bytes.NewBuffer(nil)
		if err := signozDockerfileTemplate.Execute(dockerfileBuf, config); err != nil {
			return nil, fmt.Errorf("signoz dockerfile: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(dockerfileBuf.Bytes(), filepath.Join(casting.DeploymentDir, "signoz/Dockerfile")))
		railwayBuf := bytes.NewBuffer(nil)
		if err := railwaySignozTemplate.Execute(railwayBuf, config); err != nil {
			return nil, fmt.Errorf("signoz railway.json: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(railwayBuf.Bytes(), filepath.Join(casting.DeploymentDir, "signoz/railway.json")))
	}

	// TelemetryStore migrator: Dockerfile + railway.json
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		dockerfileBuf := bytes.NewBuffer(nil)
		if err := telemetryStoreMigratorDockerfileTemplate.Execute(dockerfileBuf, config); err != nil {
			return nil, fmt.Errorf("telemetrystore-migrator dockerfile: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(dockerfileBuf.Bytes(), filepath.Join(casting.DeploymentDir, "telemetrystore-migrator/Dockerfile")))
		railwayBuf := bytes.NewBuffer(nil)
		if err := railwayTelemetryStoreMigratorTemplate.Execute(railwayBuf, config); err != nil {
			return nil, fmt.Errorf("telemetrystore-migrator railway.json: %w", err)
		}
		materials = append(materials, domain.NewBlobMaterial(railwayBuf.Bytes(), filepath.Join(casting.DeploymentDir, "telemetrystore-migrator/railway.json")))
	}

	return materials, nil
}

func (c *railwayTemplateCasting) Cast(ctx context.Context, config installation.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Please use the template.")
	return nil
}

func getRailwayMaterial(config *installation.Casting) ([]domain.StructuredMaterial, error) {
	var materials []domain.StructuredMaterial

	keeperBuf := bytes.NewBuffer(nil)
	if err := telemetryKeeperOverrideTemplate.Execute(keeperBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute keeper override template: %w", err)
	}
	keeperMaterial, err := domain.NewYAMLMaterial(keeperBuf.Bytes(), "keeper_overrides.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create keeper override material: %w", err)
	}
	materials = append(materials, keeperMaterial)

	storeBuf := bytes.NewBuffer(nil)
	if err := telemetryStoreOverrideTemplate.Execute(storeBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute keeper override template: %w", err)
	}
	storeMaterial, err := domain.NewYAMLMaterial(storeBuf.Bytes(), "store_overrides.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create keeper override material: %w", err)
	}
	materials = append(materials, storeMaterial)
	return materials, nil
}
