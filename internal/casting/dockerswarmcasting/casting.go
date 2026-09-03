package dockerswarmcasting

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/dockerswarmtooler"
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

func (casting *dockerSwarmCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newDockerSwarmMoldingEnricher(config)
}

func (casting *dockerSwarmCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {

	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to execute compose yaml template")
	}

	composeMaterial, err := domain.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create compose yaml material")
	}

	materials := []domain.Material{composeMaterial}

	for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create telemetrykeeper config material")
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create telemetrystore config material")
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "metastore", config.Spec.MetaStore.Kind.String(), filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create metastore config material")
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.Signoz.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "signoz", filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create signoz config material")
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.Ingester.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "ingester", filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create ingester config material")
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func (casting *dockerSwarmCasting) Cast(ctx context.Context, config installation.Casting, outputPath string, toolers []tooler.Tooler) error {
	swarm, err := dockerswarmtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := dockerswarmtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		File:    filepath.Join(outputPath, rootcasting.DeploymentDir, strings.TrimSuffix(composeYAMLTemplate.Name(), ".gotmpl")),
	}

	return swarm.Up(ctx, release)
}

// Melt removes the stack's services and networks; the volumes holding
// component data stay.
func (casting *dockerSwarmCasting) Melt(ctx context.Context, config installation.Casting, outputPath string, toolers []tooler.Tooler) error {
	swarm, err := dockerswarmtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := dockerswarmtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		File:    filepath.Join(outputPath, rootcasting.DeploymentDir, strings.TrimSuffix(composeYAMLTemplate.Name(), ".gotmpl")),
	}

	return swarm.Down(ctx, release)
}

func getComposeMaterial(config *installation.Casting, path string) (domain.StructuredMaterial, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to execute compose yaml template")
	}

	return domain.NewYAMLMaterial(buf.Bytes(), path)
}
