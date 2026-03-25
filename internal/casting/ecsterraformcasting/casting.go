package ecsterraformcasting

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
	return newEcsMoldingEnricher(config)
}

func (c *ecsCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	var materials []types.Material

	deployDir := rootcasting.DeploymentDir
	moduleDir := filepath.Join(deployDir, "module")

	// Root Terraform files
	rootTemplates := map[string]*types.Template{
		"main.tf.json":          mainTF,
		"variables.tf.json":     variablesTF,
		"terraform.tfvars.json": tfarsTF,
	}
	for filename, tmpl := range rootTemplates {
		m, err := executeTemplate(tmpl, config, filepath.Join(deployDir, filename))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// Module shared files
	moduleTemplates := map[string]*types.Template{
		"main.tf.json":      moduleMainTF,
		"variables.tf.json": moduleVariablesTF,
		"outputs.tf.json":   moduleOutputsTF,
	}
	for filename, tmpl := range moduleTemplates {
		m, err := executeTemplate(tmpl, config, filepath.Join(moduleDir, filename))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// TelemetryKeeper
	if config.Spec.TelemetryKeeper.Spec.Enabled {
		m, err := executeTemplate(moduleTelemetryKeeperTF, config, filepath.Join(moduleDir, "telemetrykeeper.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
			material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(moduleDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename))
			if err != nil {
				return nil, err
			}
			materials = append(materials, material)
		}
	}

	// TelemetryStore
	if config.Spec.TelemetryStore.Spec.Enabled {
		m, err := executeTemplate(moduleTelemetryStoreTF, config, filepath.Join(moduleDir, "telemetrystore.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
			material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(moduleDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename))
			if err != nil {
				return nil, err
			}
			materials = append(materials, material)
		}
	}

	// TelemetryStore migrator
	if config.Spec.TelemetryStore.Spec.Enabled {
		m, err := executeTemplate(moduleMigratorTF, config, filepath.Join(moduleDir, "telemetrystore_migrator.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// MetaStore
	if config.Spec.MetaStore.Spec.Enabled {
		m, err := executeTemplate(moduleMetaStoreTF, config, filepath.Join(moduleDir, "metastore.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
			material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(moduleDir, "metastore", config.Spec.MetaStore.Kind.String(), filename))
			if err != nil {
				return nil, err
			}
			materials = append(materials, material)
		}
	}

	// Signoz
	if config.Spec.Signoz.Spec.Enabled {
		m, err := executeTemplate(moduleSignozTF, config, filepath.Join(moduleDir, "signoz.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// Ingester
	if config.Spec.Ingester.Spec.Enabled {
		m, err := executeTemplate(moduleIngesterTF, config, filepath.Join(moduleDir, "ingester.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.Ingester.Spec.Config.Data {
			material, err := types.NewYAMLMaterial([]byte(content), filepath.Join(moduleDir, "ingester", filename))
			if err != nil {
				return nil, err
			}
			materials = append(materials, material)
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

// executeTemplate renders a template and returns a JSONMaterial at the given path.
func executeTemplate(tmpl *types.Template, config v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, config); err != nil {
		return types.Material{}, fmt.Errorf("failed to execute template for %s: %w", path, err)
	}
	return types.NewJSONMaterial(buf.Bytes(), path)
}

// getMaterials renders all module templates and returns them as JSONMaterials.
func getMaterials(config *v1alpha1.Casting) ([]types.Material, error) {
	var materials []types.Material

	for _, tmpl := range []*types.Template{
		moduleMainTF,
		moduleTelemetryStoreTF,
		moduleTelemetryKeeperTF,
		moduleMetaStoreTF,
		moduleSignozTF,
		moduleIngesterTF,
	} {
		m, err := executeTemplate(tmpl, *config, tmpl.GetPath())
		if err != nil {
			return nil, fmt.Errorf("failed to create material: %w", err)
		}
		materials = append(materials, m)
	}

	return materials, nil
}
