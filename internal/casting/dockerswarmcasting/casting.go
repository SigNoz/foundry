package dockerswarmcasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
)

var _ rootcasting.Casting = (*dockerSwarmCasting)(nil)

type dockerSwarmCasting struct {
	logger   *slog.Logger
	castings []*domain.Template
}

func New(logger *slog.Logger) *dockerSwarmCasting {
	return &dockerSwarmCasting{
		logger: logger,
		castings: []*domain.Template{
			composeYAMLTemplate,
		},
	}
}

func (casting *dockerSwarmCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newDockerSwarmMoldingEnricher(config)
}

func (casting *dockerSwarmCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]domain.Material, error) {

	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute compose yaml template: %w", err)
	}

	composeMaterial, err := domain.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to create compose yaml material: %w", err)
	}

	materials := []domain.Material{composeMaterial}

	for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create telemetrykeeper config material: %w", err)
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create telemetrystore config material: %w", err)
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "metastore", config.Spec.MetaStore.Kind.String(), filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create metastore config material: %w", err)
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.Signoz.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "signoz", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create signoz config material: %w", err)
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.Ingester.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "ingester", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create ingester config material: %w", err)
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func (casting *dockerSwarmCasting) Cast(ctx context.Context, config v1alpha1.Casting, outputPath string) error {
	casting.logger.InfoContext(ctx, "Deploying stack to Docker Swarm")

	composeFile := filepath.Join(outputPath, rootcasting.DeploymentDir, "compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file does not exist at path: %s", composeFile)
	}

	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := []string{"stack", "deploy", "-d", "-c", composeFile}

	args = append(args, config.Metadata.Name)

	casting.logger.DebugContext(runctx, "Running command", slog.String("command", strings.Join(append([]string{"docker"}, args...), " ")))

	cmd := exec.CommandContext(runctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		casting.logger.ErrorContext(runctx, "Stack deploy failed", slog.String("error", err.Error()))
		return err
	}

	casting.logger.InfoContext(runctx, "Stack deployed successfully")

	return nil
}

func getComposeMaterial(config *v1alpha1.Casting, path string) (domain.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return domain.Material{}, fmt.Errorf("failed to execute compose yaml template: %w", err)
	}

	return domain.NewYAMLMaterial(buf.Bytes(), path)
}
