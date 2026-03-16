package azureacacasting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

const (
	annotationResourceGroup  = "foundry.signoz.io/aca-resource-group"
	annotationEnvironment    = "foundry.signoz.io/aca-environment"
	annotationSubscriptionID = "foundry.signoz.io/aca-subscription-id"
)

var _ rootcasting.Casting = (*acaCasting)(nil)

type acaCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

type acaConfig struct {
	resourceGroup  string
	environment    string
	subscriptionID string
}

func newAcaConfig(annotations map[string]string) (acaConfig, error) {
	required := []string{annotationResourceGroup, annotationEnvironment, annotationSubscriptionID}

	if annotations == nil {
		return acaConfig{}, fmt.Errorf("metadata.annotations is required, set %s",
			strings.Join(required, " and "))
	}

	for _, r := range required {
		val, ok := annotations[r]
		if !ok || val == "" {
			return acaConfig{}, fmt.Errorf("required annotation %q is missing", r)
		}
	}

	return acaConfig{
		resourceGroup:  annotations[annotationResourceGroup],
		environment:    annotations[annotationEnvironment],
		subscriptionID: annotations[annotationSubscriptionID],
	}, nil
}

// ACAKnobs defines the platform-specific knobs supported by the Azure
// Container Apps casting. These are per-component and live under
// spec.<component>.spec.config.knobs.
type ACAKnobs struct {
	// Resources contains the container resource allocation (cpu and memory).
	Resources map[string]any `json:"resources,omitempty" yaml:"resources,omitempty" description:"Container resource allocation (cpu, memory)"`
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
	if err := validateKnobs(cfg); err != nil {
		return nil, fmt.Errorf("invalid aca knobs: %w", err)
	}

	var materials []types.Material

	// Render container app YAML specs (config is embedded as secrets in the templates).
	for _, tmpl := range c.castings {
		m, err := c.forgeCasting(tmpl, &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to forge: %w", err)
		}
		materials = append(materials, m...)
	}

	return materials, nil
}

func (c *acaCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Deploying to Azure Container Apps")

	deploymentDir := filepath.Join(poursPath, rootcasting.DeploymentDir)
	if _, err := os.Stat(filepath.Join(deploymentDir, "telemetrykeeper", "containerapp.yaml")); os.IsNotExist(err) {
		return fmt.Errorf("forged files do not exist at %s, run 'forge' first", deploymentDir)
	}

	cfg, err := newAcaConfig(config.Metadata.Annotations)
	if err != nil {
		return err
	}

	runctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Deploy in dependency order.
	type step struct {
		name     string
		yamlPath string
		isJob    bool
	}

	steps := []step{
		{name: config.Metadata.Name + "-clickhouse-keeper-0", yamlPath: "telemetrykeeper/containerapp.yaml"},
		{name: config.Metadata.Name + "-clickhouse", yamlPath: "telemetrystore/containerapp.yaml"},
		{name: config.Metadata.Name + "-telemetrystore-migrator", yamlPath: "telemetrystore-migrator/job.yaml", isJob: true},
		{name: config.Metadata.Name + "-signoz", yamlPath: "signoz/containerapp.yaml"},
		{name: config.Metadata.Name + "-ingester", yamlPath: "ingester/containerapp.yaml"},
	}

	for _, s := range steps {
		fullPath := filepath.Join(deploymentDir, s.yamlPath)
		c.logger.InfoContext(runctx, "deploying", slog.String("name", s.name))

		if s.isJob {
			if err := c.azContainerAppJobCreate(runctx, cfg, s.name, fullPath); err != nil {
				return fmt.Errorf("failed to deploy job %q: %w", s.name, err)
			}
			if err := c.azContainerAppJobStart(runctx, cfg, s.name); err != nil {
				return fmt.Errorf("failed to start job %q: %w", s.name, err)
			}
		} else {
			if err := c.azContainerAppCreate(runctx, cfg, s.name, fullPath); err != nil {
				return fmt.Errorf("failed to deploy container app %q: %w", s.name, err)
			}
		}
	}

	c.logger.InfoContext(ctx, "Deployment complete",
		slog.String("name", config.Metadata.Name),
		slog.String("resource_group", cfg.resourceGroup),
	)
	return nil
}

func (c *acaCasting) azContainerAppCreate(ctx context.Context, cfg acaConfig, name, yamlPath string) error {
	args := []string{
		"containerapp", "create",
		"--subscription", cfg.subscriptionID,
		"--name", name,
		"--resource-group", cfg.resourceGroup,
		"--environment", cfg.environment,
		"--yaml", yamlPath,
		"--output", "none",
	}

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "az "+strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, "az", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("az containerapp create failed: %w", err)
	}

	c.logger.InfoContext(ctx, "deployed container app", slog.String("name", name))
	return nil
}

func (c *acaCasting) azContainerAppJobCreate(ctx context.Context, cfg acaConfig, name, yamlPath string) error {
	args := []string{
		"containerapp", "job", "create",
		"--subscription", cfg.subscriptionID,
		"--name", name,
		"--resource-group", cfg.resourceGroup,
		"--environment", cfg.environment,
		"--yaml", yamlPath,
		"--output", "none",
	}

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "az "+strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, "az", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("az containerapp job create failed: %w", err)
	}

	c.logger.InfoContext(ctx, "deployed job", slog.String("name", name))
	return nil
}

func (c *acaCasting) azContainerAppJobStart(ctx context.Context, cfg acaConfig, name string) error {
	args := []string{
		"containerapp", "job", "start",
		"--subscription", cfg.subscriptionID,
		"--name", name,
		"--resource-group", cfg.resourceGroup,
		"--output", "none",
	}

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "az "+strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, "az", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("az containerapp job start failed: %w", err)
	}

	c.logger.InfoContext(ctx, "job execution started", slog.String("name", name))
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
