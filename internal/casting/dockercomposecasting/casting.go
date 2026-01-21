package dockercomposecasting

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
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

func (casting *dockerComposeCasting) Cast(ctx context.Context, config v1alpha1.Casting, outputPath string) error {
	casting.logger.InfoContext(ctx, "Executing commands for platform")

	// Check if compose file exists
	composeFile := filepath.Join(outputPath, "compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file does not exist at path: %s", composeFile)
	}

	// Create a context with 5-minute timeout
	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Get the available docker compose command
	composeCmd, err := getComposeCommand(runctx)
	if err != nil {
		casting.logger.ErrorContext(runctx, "Docker compose not available", slog.String("error", err.Error()))
		return fmt.Errorf("docker compose not available: %w", err)
	}

	// Build command arguments: "up -d"
	args := append(composeCmd[1:], "-f", composeFile, "up", "-d")

	casting.logger.DebugContext(runctx, "Running command", slog.String("command", strings.Join(append([]string{composeCmd[0]}, args...), " ")))

	cmd := exec.CommandContext(runctx, composeCmd[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
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

// getComposeCommand detects the available docker compose command.
// It checks for "docker compose" (newer, preferred) first, then falls back to "docker-compose" (legacy).
func getComposeCommand(ctx context.Context) ([]string, error) {
	// Check "docker compose" first (newer, preferred)
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.CommandContext(ctx, "docker", "compose", "version")
		if err := cmd.Run(); err == nil {
			return []string{"docker", "compose"}, nil
		}
	}

	// Fallback to "docker-compose" (legacy)
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}, nil
	}

	return nil, errors.New("neither 'docker compose' nor 'docker-compose' is available")
}
