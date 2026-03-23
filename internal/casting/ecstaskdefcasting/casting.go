package ecstaskdefcasting

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
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ rootcasting.Casting = (*ecsCasting)(nil)

type ecsCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *ecsCasting {
	return &ecsCasting{
		logger: logger,
	}
}

func (c *ecsCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newEcsMoldingEnricher(), nil
}

// executeTF renders a template and returns a TextMaterial at the given path.
func executeTF(tmpl *types.Template, config v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, config); err != nil {
		return types.Material{}, fmt.Errorf("failed to execute template for %s: %w", path, err)
	}
	return types.NewTextMaterial(buf.Bytes(), path), nil
}

// executeJSON renders a template and returns a JSONMaterial at the given path.
func executeJSON(tmpl *types.Template, config v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, config); err != nil {
		return types.Material{}, fmt.Errorf("failed to execute template for %s: %w", path, err)
	}
	return types.NewJSONMaterial(buf.Bytes(), path)
}

func (c *ecsCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	var materials []types.Material

	deployDir := rootcasting.DeploymentDir
	moduleDir := filepath.Join(deployDir, "module")
	configsDir := filepath.Join(moduleDir, "configs")

	// Root Terraform files
	rootTemplates := map[string]*types.Template{
		"main.tf.json":      rootMainTF,
		"variables.tf.json": rootVariablesTF,
	}
	for filename, tmpl := range rootTemplates {
		m, err := executeTF(tmpl, config, filepath.Join(deployDir, filename))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// Root terraform.tfvars.json (JSONMaterial for patchability)
	tfvars, err := executeJSON(rootTfvarsTF, config, filepath.Join(deployDir, "terraform.tfvars.json"))
	if err != nil {
		return nil, err
	}
	materials = append(materials, tfvars)

	// Module shared files
	moduleTemplates := map[string]*types.Template{
		"main.tf.json":      moduleMainTF,
		"variables.tf.json": moduleVariablesTF,
		"outputs.tf.json":   moduleOutputsTF,
	}
	for filename, tmpl := range moduleTemplates {
		m, err := executeTF(tmpl, config, filepath.Join(moduleDir, filename))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// TelemetryKeeper
	if config.Spec.TelemetryKeeper.Spec.Enabled {
		m, err := executeTF(moduleTelemetryKeeperTF, config, filepath.Join(moduleDir, "telemetrykeeper.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
			materials = append(materials, types.NewTextMaterial([]byte(content),
				filepath.Join(configsDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename)))
		}
	}

	// TelemetryStore
	if config.Spec.TelemetryStore.Spec.Enabled {
		m, err := executeTF(moduleTelemetryStoreTF, config, filepath.Join(moduleDir, "telemetrystore.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
			materials = append(materials, types.NewTextMaterial([]byte(content),
				filepath.Join(configsDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename)))
		}
	}

	// TelemetryStore migrator
	if config.Spec.TelemetryStore.Spec.Enabled {
		m, err := executeTF(moduleMigratorTF, config, filepath.Join(moduleDir, "telemetrystore_migrator.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// MetaStore
	if config.Spec.MetaStore.Spec.Enabled {
		m, err := executeTF(moduleMetaStoreTF, config, filepath.Join(moduleDir, "metastore.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
			materials = append(materials, types.NewTextMaterial([]byte(content),
				filepath.Join(configsDir, "metastore", config.Spec.MetaStore.Kind.String(), filename)))
		}
	}

	// Signoz
	if config.Spec.Signoz.Spec.Enabled {
		m, err := executeTF(moduleSignozTF, config, filepath.Join(moduleDir, "signoz.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// Ingester
	if config.Spec.Ingester.Spec.Enabled {
		m, err := executeTF(moduleIngesterTF, config, filepath.Join(moduleDir, "ingester.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.Ingester.Spec.Config.Data {
			materials = append(materials, types.NewTextMaterial([]byte(content),
				filepath.Join(configsDir, "ingester", filename)))
		}
	}

	return materials, nil
}

func (c *ecsCasting) Cast(ctx context.Context, config v1alpha1.Casting, outputPath string) error {
	c.logger.InfoContext(ctx, "Running Terraform for ECS deployment")

	deploymentDir := filepath.Join(outputPath, rootcasting.DeploymentDir)

	// Verify terraform files exist
	if _, err := os.Stat(filepath.Join(deploymentDir, "main.tf.json")); os.IsNotExist(err) {
		return fmt.Errorf("terraform files do not exist at path: %s; run forge first", deploymentDir)
	}

	// Create a context with 10-minute timeout (terraform can be slow)
	runctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Run terraform init
	c.logger.InfoContext(runctx, "Running terraform init")
	initCmd := exec.CommandContext(runctx, "terraform", "-chdir="+deploymentDir, "init")
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		c.logger.ErrorContext(runctx, "terraform init failed", slog.String("error", err.Error()))
		return fmt.Errorf("terraform init failed: %w", err)
	}

	// Run terraform apply
	c.logger.InfoContext(runctx, "Running terraform apply")
	args := []string{"-chdir=" + deploymentDir, "apply", "-auto-approve"}
	c.logger.DebugContext(runctx, "Running command", slog.String("command", "terraform "+strings.Join(args, " ")))

	applyCmd := exec.CommandContext(runctx, "terraform", args...)
	applyCmd.Stdout = os.Stdout
	applyCmd.Stderr = os.Stderr
	if err := applyCmd.Run(); err != nil {
		c.logger.ErrorContext(runctx, "terraform apply failed", slog.String("error", err.Error()))
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	c.logger.InfoContext(runctx, "Terraform apply completed successfully")
	return nil
}
