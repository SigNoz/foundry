package dockercomposecasting

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/dockercomposetooler"
)

var _ rootcasting.Casting = (*dockerComposeCasting)(nil)

type dockerComposeCasting struct {
	logger   *slog.Logger
	castings []*domain.Template
}

func New(logger *slog.Logger) *dockerComposeCasting {
	return &dockerComposeCasting{
		logger: logger,
		castings: []*domain.Template{
			composeYAMLTemplate,
		},
	}
}

func (casting *dockerComposeCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newDockerComposeMoldingEnricher(casting.logger, config)
}

func (casting *dockerComposeCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute compose yaml template")
	}

	composeMaterial, err := domain.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create compose yaml material")
	}

	materials := []domain.Material{composeMaterial}

	// Add telemetrykeeper config files
	for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create telemetrykeeper config material")
		}
		materials = append(materials, material)
	}

	// Add telemetrystore config files
	for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create telemetrystore config material")
		}
		materials = append(materials, material)
	}

	// Add metastore config files
	for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "metastore", config.Spec.MetaStore.Kind.String(), filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create metastore config material")
		}
		materials = append(materials, material)
	}

	// Add signoz config files
	for filename, content := range config.Spec.Signoz.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "signoz", filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create signoz config material")
		}
		materials = append(materials, material)
	}

	// Add ingester config files
	for filename, content := range config.Spec.Ingester.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "ingester", filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create ingester config material")
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func (casting *dockerComposeCasting) Cast(ctx context.Context, config installation.Casting, outputPath string, toolers []tooler.Tooler) error {
	compose, err := dockercomposetooler.Lookup(toolers)
	if err != nil {
		return err
	}
	release := dockercomposetooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		File:    filepath.Join(outputPath, rootcasting.DeploymentDir, strings.TrimSuffix(composeYAMLTemplate.Name(), ".gotmpl")),
	}

	return compose.Up(ctx, release)
}

// Melt removes the containers and networks; the volumes holding component
// data stay.
func (casting *dockerComposeCasting) Melt(ctx context.Context, config installation.Casting, outputPath string, toolers []tooler.Tooler) error {
	compose, err := dockercomposetooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := dockercomposetooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		File:    filepath.Join(outputPath, rootcasting.DeploymentDir, strings.TrimSuffix(composeYAMLTemplate.Name(), ".gotmpl")),
	}

	return compose.Down(ctx, release)
}

func getComposeMaterial(config *installation.Casting, path string) (domain.StructuredMaterial, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute compose yaml template")
	}

	return domain.NewYAMLMaterial(buf.Bytes(), path)
}
