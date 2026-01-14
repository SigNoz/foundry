package linuxcasting

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

var _ casting.Casting = (*linuxCasting)(nil)

type linuxCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

type serviceMaterials struct {
	signoz          types.Material
	ingester        types.Material
	telemetryStore  types.Material
	telemetryKeeper types.Material
	metaStore       types.Material
}

// serviceDefinition pairs a template with its output path
type serviceDefinition struct {
	template *types.Template
	path string
}

func New(logger *slog.Logger) *linuxCasting {
	return &linuxCasting{
		logger: logger,
		castings: []*types.Template{
			telemetryKeeperServiceTemplate,
			telemetryStoreServiceTemplate,
			metaStoreServiceTemplate,
			signozServiceTemplate,
			ingesterServiceTemplate,
		},
	}
}

func (casting *linuxCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newLinuxMoldingEnricher(config)
}

func (casting *linuxCasting) Forge(ctx context.Context, config v1alpha1.Casting) ([]types.Material, error) {
	// Get generated service template materials (only enabled ones)
	materials, err := getServiceMaterials(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to create template materials: %w", err)
	}

	// Collect config data from all components
	componentConfigs := []*v1alpha1.TypeConfig{
		&config.Spec.Signoz.Spec.Config,
		&config.Spec.Ingester.Spec.Config,
		&config.Spec.TelemetryStore.Spec.Config,
		&config.Spec.TelemetryKeeper.Spec.Config,
		&config.Spec.MetaStore.Spec.Config,
	}

	for _, cfg := range componentConfigs {
		for name, content := range cfg.Data {
			m, err := types.NewYAMLMaterial([]byte(content), name)
			if err != nil {
				return nil, fmt.Errorf("failed to create material for %s: %w", name, err)
			}
			materials = append(materials, m)
		}
	}

	return materials, nil
}

func (casting *linuxCasting) Cast(ctx context.Context, config v1alpha1.Casting) error {
	casting.logger.InfoContext(ctx, "Executing commands for platform")

	// Create a context with 5-minute timeout
	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

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

func getServiceMaterials(config *v1alpha1.Casting) ([]types.Material, error) {
	materials := []types.Material{}

	// Define all services with their enabled status
	services := []struct {
		enabled  bool
		template *types.Template
		path string
	}{
		{config.Spec.Signoz.Spec.Enabled, signozServiceTemplate, "signoz.service"},
		{config.Spec.Ingester.Spec.Enabled, ingesterServiceTemplate, "ingester.service"},
		{config.Spec.TelemetryStore.Spec.Enabled, telemetryStoreServiceTemplate, "telemetrystore.service"},
		{config.Spec.TelemetryKeeper.Spec.Enabled, telemetryKeeperServiceTemplate, "telemetrykeeper.service"},
		{config.Spec.MetaStore.Spec.Enabled, metaStoreServiceTemplate, "metastore.service"},
	}

	for _, svc := range services {
		if !svc.enabled {
			continue
		}

		material, err := executeServiceTemplate(*svc.template, config, svc.path)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s material: %w", svc.path, err)
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func executeServiceTemplate(template types.Template, config *v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := template.Execute(buf, config)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to execute template: %w", err)
	}

	return types.NewSystemdMaterial(buf.Bytes(), path)
}