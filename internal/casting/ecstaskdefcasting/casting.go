package ecstaskdefcasting

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

var _ rootcasting.Casting = (*ecsCasting)(nil)

type ecsCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

var castings = []*types.Template{
	telemetryKeeperTaskDefinition,
	telemetryStoreTaskDefinition,
	migratorTaskDefinition,
	metaStoreTaskDefinition,
	signozTaskDefinition,
	ingesterTaskDefinition,
}

func New(logger *slog.Logger) *ecsCasting {
	return &ecsCasting{
		logger:   logger,
		castings: castings,
	}
}

func (c *ecsCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newEcsMoldingEnricher(), nil
}

func (c *ecsCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	var materials []types.Material

	// TelemetryKeeper: task definition + configs
	if config.Spec.TelemetryKeeper.Spec.Enabled {
		buf := bytes.NewBuffer(nil)
		if err := telemetryKeeperTaskDefinition.Execute(buf, config); err != nil {
			return nil, fmt.Errorf("failed to execute telemetrykeeper task definition template: %w", err)
		}
		materials = append(materials, types.NewTextMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), "task-definition.json")))

		for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
			materials = append(materials, types.NewTextMaterial([]byte(content),
				filepath.Join(rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename)))
		}
	}

	// TelemetryStore: task definition + configs
	if config.Spec.TelemetryStore.Spec.Enabled {
		buf := bytes.NewBuffer(nil)
		if err := telemetryStoreTaskDefinition.Execute(buf, config); err != nil {
			return nil, fmt.Errorf("failed to execute telemetrystore task definition template: %w", err)
		}
		materials = append(materials, types.NewTextMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), "task-definition.json")))

		for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
			materials = append(materials, types.NewTextMaterial([]byte(content),
				filepath.Join(rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename)))
		}
	}

	// TelemetryStore migrator: task definition
	if config.Spec.TelemetryStore.Spec.Enabled {
		buf := bytes.NewBuffer(nil)
		if err := migratorTaskDefinition.Execute(buf, config); err != nil {
			return nil, fmt.Errorf("failed to execute telemetrystore-migrator task definition template: %w", err)
		}
		materials = append(materials, types.NewTextMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "telemetrystore-migrator/task-definition.json")))
	}

	// MetaStore: task definition + configs
	if config.Spec.MetaStore.Spec.Enabled {
		buf := bytes.NewBuffer(nil)
		if err := metaStoreTaskDefinition.Execute(buf, config); err != nil {
			return nil, fmt.Errorf("failed to execute metastore task definition template: %w", err)
		}
		materials = append(materials, types.NewTextMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "metastore", config.Spec.MetaStore.Kind.String(), "task-definition.json")))

		for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
			materials = append(materials, types.NewTextMaterial([]byte(content),
				filepath.Join(rootcasting.DeploymentDir, "metastore", config.Spec.MetaStore.Kind.String(), filename)))
		}
	}

	// Signoz: task definition
	if config.Spec.Signoz.Spec.Enabled {
		buf := bytes.NewBuffer(nil)
		if err := signozTaskDefinition.Execute(buf, config); err != nil {
			return nil, fmt.Errorf("failed to execute signoz task definition template: %w", err)
		}
		materials = append(materials, types.NewTextMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "signoz/task-definition.json")))
	}

	// Ingester: task definition + configs
	if config.Spec.Ingester.Spec.Enabled {
		buf := bytes.NewBuffer(nil)
		if err := ingesterTaskDefinition.Execute(buf, config); err != nil {
			return nil, fmt.Errorf("failed to execute ingester task definition template: %w", err)
		}
		materials = append(materials, types.NewTextMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "ingester/task-definition.json")))

		for filename, content := range config.Spec.Ingester.Spec.Config.Data {
			materials = append(materials, types.NewTextMaterial([]byte(content),
				filepath.Join(rootcasting.DeploymentDir, "ingester", filename)))
		}
	}

	return materials, nil
}

func (c *ecsCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	return fmt.Errorf("Cast is not supported for ECS task definition casting; Please run Forge first and checkout docs/examples/ecs/ec2/README.md for manual deployment steps")
}
