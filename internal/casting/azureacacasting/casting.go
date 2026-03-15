package azureacacasting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ rootcasting.Casting = (*acaCasting)(nil)

type acaCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

// ACAKnobs defines the platform-specific knobs supported by the Azure
// Container Apps casting. These are per-component and live under
// spec.<component>.spec.config.knobs.
type ACAKnobs struct {
	// Resources contains the container resource allocation (cpu and memory).
	Resources map[string]any `json:"resources,omitempty" yaml:"resources,omitempty" description:"Container resource allocation (cpu, memory)"`

	// Scale contains the scaling configuration (minReplicas, maxReplicas).
	Scale map[string]any `json:"scale,omitempty" yaml:"scale,omitempty" description:"Scaling configuration (minReplicas, maxReplicas)"`
}

func New(logger *slog.Logger) *acaCasting {
	return &acaCasting{
		logger: logger,
		castings: []*types.Template{
			telemetrykeeperContainerapp,
			telemetrystoreContainerapp,
			signozContainerapp,
			ingesterContainerapp,
			telemetrystoreMigratorJob,
		},
	}
}

func (c *acaCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newAcaMoldingEnricher(config), nil
}

func (c *acaCasting) Forge(ctx context.Context, cfg v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	// Validate knobs before rendering templates so users get clear errors on
	// unknown keys or type mismatches.
	if err := validateKnobs(cfg); err != nil {
		return nil, fmt.Errorf("invalid aca knobs: %w", err)
	}

	var materials []types.Material

	// Render container app YAML specs.
	for _, tmpl := range c.castings {
		m, err := c.forgeCasting(tmpl, &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to forge: %w", err)
		}
		materials = append(materials, m...)
	}

	// Add telemetrykeeper config files.
	for filename, content := range cfg.Spec.TelemetryKeeper.Spec.Config.Data {
		material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrykeeper", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create telemetrykeeper config material: %w", err)
		}
		materials = append(materials, material)
	}

	// Add telemetrystore config files.
	for filename, content := range cfg.Spec.TelemetryStore.Spec.Config.Data {
		material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrystore", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create telemetrystore config material: %w", err)
		}
		materials = append(materials, material)
	}

	// Add ingester config files.
	for filename, content := range cfg.Spec.Ingester.Spec.Config.Data {
		material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "ingester", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create ingester config material: %w", err)
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func (c *acaCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Please run 'forge' first to generate the Azure Container Apps deployment files",
		slog.String("pours_path", poursPath))
	return nil
}

func (c *acaCasting) forgeCasting(tmpl *types.Template, cfg *v1alpha1.Casting) ([]types.Material, error) {
	templatePath := tmpl.GetPath()
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", templatePath, err)
	}
	relPath := strings.TrimPrefix(templatePath, "templates/")
	relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))
	path := filepath.Join(rootcasting.DeploymentDir, relPath)
	material, err := types.NewYAMLMaterial(buf.Bytes(), path)
	if err != nil {
		return nil, fmt.Errorf("create material %s: %w", templatePath, err)
	}
	return []types.Material{material}, nil
}

// validateKnobs ensures that all per-component knobs conform to ACAKnobs.
// This catches type mismatches and unknown fields before templates run.
func validateKnobs(cfg v1alpha1.Casting) error {
	components := map[string]map[string]any{
		"signoz":          cfg.Spec.Signoz.Spec.Config.Knobs,
		"ingester":        cfg.Spec.Ingester.Spec.Config.Knobs,
		"telemetrystore":  cfg.Spec.TelemetryStore.Spec.Config.Knobs,
		"telemetrykeeper": cfg.Spec.TelemetryKeeper.Spec.Config.Knobs,
	}

	for component, knobs := range components {
		if knobs == nil {
			continue
		}

		data, err := json.Marshal(knobs)
		if err != nil {
			return fmt.Errorf("component %s: failed to marshal knobs: %w", component, err)
		}

		var k ACAKnobs
		if err := json.Unmarshal(data, &k); err != nil {
			return fmt.Errorf("component %s: %w", component, err)
		}
	}
	return nil
}
