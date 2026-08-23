package ecsterraformcasting

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
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

func (c *ecsCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newEcsMoldingEnricher(config)
}

func (c *ecsCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {
	var materials []domain.Material

	deployDir := rootcasting.DeploymentDir
	moduleDir := filepath.Join(deployDir, "module")

	// Root Terraform files
	rootTemplates := map[string]*domain.Template{
		"main.tf.json":          mainTF,
		"variables.tf.json":     variablesTF,
		"terraform.tfvars.json": tfarsTF,
	}
	for filename, tmpl := range rootTemplates {
		m, err := tmpl.Render(config, filepath.Join(deployDir, filename))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// Module shared files
	moduleTemplates := map[string]*domain.Template{
		"main.tf.json":      moduleMainTF,
		"variables.tf.json": moduleVariablesTF,
		"outputs.tf.json":   moduleOutputsTF,
	}
	for filename, tmpl := range moduleTemplates {
		m, err := tmpl.Render(config, filepath.Join(moduleDir, filename))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// TelemetryKeeper
	if config.Spec.TelemetryKeeper.Spec.IsEnabled() {
		m, err := moduleTelemetryKeeperTF.Render(config, filepath.Join(moduleDir, "telemetrykeeper.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
			material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(moduleDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename))
			if err != nil {
				return nil, err
			}
			materials = append(materials, material)
		}
	}

	// TelemetryStore
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		m, err := moduleTelemetryStoreTF.Render(config, filepath.Join(moduleDir, "telemetrystore.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
			material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(moduleDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename))
			if err != nil {
				return nil, err
			}
			materials = append(materials, material)
		}
	}

	// TelemetryStore migrator
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		m, err := moduleMigratorTF.Render(config, filepath.Join(moduleDir, "telemetrystore_migrator.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// MetaStore
	if config.Spec.MetaStore.Spec.IsEnabled() {
		m, err := moduleMetaStoreTF.Render(config, filepath.Join(moduleDir, "metastore.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
			material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(moduleDir, "metastore", config.Spec.MetaStore.Kind.String(), filename))
			if err != nil {
				return nil, err
			}
			materials = append(materials, material)
		}
	}

	// Signoz
	if config.Spec.Signoz.Spec.IsEnabled() {
		m, err := moduleSignozTF.Render(config, filepath.Join(moduleDir, "signoz.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}

	// Ingester
	if config.Spec.Ingester.Spec.IsEnabled() {
		m, err := moduleIngesterTF.Render(config, filepath.Join(moduleDir, "ingester.tf.json"))
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)

		for filename, content := range config.Spec.Ingester.Spec.Config.Data {
			material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(moduleDir, "ingester", filename))
			if err != nil {
				return nil, err
			}
			materials = append(materials, material)
		}
	}

	return materials, nil
}

func (c *ecsCasting) Cast(ctx context.Context, config installation.Casting, outputPath string, toolers []tooler.Tooler) error {
	c.logger.InfoContext(ctx, "Running Terraform for ECS deployment")

	terraform, err := terraformtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	return terraform.Apply(ctx, terraformtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Root:    filepath.Join(outputPath, rootcasting.DeploymentDir),
	})
}

// getMaterials renders all module templates and returns them as JSONMaterials.
func getMaterials(config *installation.Casting) ([]domain.StructuredMaterial, error) {
	var materials []domain.StructuredMaterial

	for _, tmpl := range []*domain.Template{
		moduleMainTF,
		moduleTelemetryStoreTF,
		moduleTelemetryKeeperTF,
		moduleMetaStoreTF,
		moduleSignozTF,
		moduleIngesterTF,
	} {
		m, err := tmpl.Render(*config, tmpl.Path())
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create material")
		}
		sm, ok := m.(domain.StructuredMaterial)
		if !ok {
			return nil, errors.Newf(errors.TypeInternal, "template %s does not produce a structured material", tmpl.Path())
		}
		materials = append(materials, sm)
	}

	return materials, nil
}

// Uncast is not implemented for this casting yet.
func (c *ecsCasting) Uncast(ctx context.Context, config installation.Casting, outputPath string, _ []tooler.Tooler) error {
	return errors.Newf(errors.TypeUnsupported, "uncast is not implemented for this casting yet")
}
