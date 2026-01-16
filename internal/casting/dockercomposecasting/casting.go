package dockercomposecasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
)

var _ casting.Casting = (*dockerComposeCasting)(nil)

type dockerComposeCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

func New(logger *slog.Logger) *dockerComposeCasting {
	return &dockerComposeCasting{
		logger: logger,
		castings: []*types.Template{
			composeYAMLTemplate,
		},
	}
}

func (casting *dockerComposeCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newDockerComposeMoldingEnricher(config)
}

func (casting *dockerComposeCasting) Forge(ctx context.Context, config v1alpha1.Casting) ([]types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute compose yaml template: %w", err)
	}

	composeMaterial, err := types.NewYAMLMaterial(buf.Bytes(), "compose.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create compose yaml material: %w", err)
	}

	materials := []types.Material{composeMaterial}

	// Add telemetrykeeper config files
	for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
		material, err := types.NewYAMLMaterial([]byte(content), fmt.Sprintf("configs/telemetrykeeper/%s", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create telemetrykeeper config material: %w", err)
		}
		materials = append(materials, material)
	}

	// Add telemetrystore config files
	for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
		material, err := types.NewYAMLMaterial([]byte(content), fmt.Sprintf("configs/telemetrystore/%s", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create telemetrystore config material: %w", err)
		}
		materials = append(materials, material)
	}

	// Add metastore config files
	for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
		material, err := types.NewYAMLMaterial([]byte(content), fmt.Sprintf("configs/metastore/%s", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create metastore config material: %w", err)
		}
		materials = append(materials, material)
	}

	// Add signoz config files
	for filename, content := range config.Spec.Signoz.Spec.Config.Data {
		material, err := types.NewYAMLMaterial([]byte(content), fmt.Sprintf("configs/signoz/%s", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create signoz config material: %w", err)
		}
		materials = append(materials, material)
	}

	// Add ingester config files
	for filename, content := range config.Spec.Ingester.Spec.Config.Data {
		material, err := types.NewYAMLMaterial([]byte(content), fmt.Sprintf("configs/ingester/%s", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create ingester config material: %w", err)
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func (casting *dockerComposeCasting) Cast(ctx context.Context, config v1alpha1.Casting) error {
	casting.logger.InfoContext(ctx, "Executing commands for platform")

	// Create a context with 5-minute timeout
	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Join commands with && to run in sequence
	//command := strings.Join(cast.Execute, " && ")
	command := ""

	casting.logger.DebugContext(runctx, "Running command", slog.String("command", command))

	cmd := exec.CommandContext(runctx, "sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		casting.logger.ErrorContext(runctx, "Command execution failed", slog.String("error", err.Error()))
		return err
	}

	casting.logger.InfoContext(runctx, "Command executed successfully")
	return nil

}

func getComposeMaterial(config *v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to execute compose yaml template: %w", err)
	}

	return types.NewYAMLMaterial(buf.Bytes(), path)
}
