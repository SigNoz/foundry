package kuberneteshelmcasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/helmtooler"
	"sigs.k8s.io/yaml"
)

var _ rootcasting.Casting = (*helmCasting)(nil)

type helmCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *helmCasting {
	return &helmCasting{logger: logger}
}

func (c *helmCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newHelmMoldingEnricher(config), nil
}

func (c *helmCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {
	buf := bytes.NewBuffer(nil)
	if err := valuesYAMLTemplate.Execute(buf, config); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to execute values yaml template")
	}

	material, err := domain.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "values.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create values yaml material")
	}

	return []domain.Material{material}, nil
}

func (c *helmCasting) Cast(ctx context.Context, config installation.Casting, poursPath string, toolers []tooler.Tooler) error {
	helm, err := helmtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release, err := c.release(config, poursPath)
	if err != nil {
		return err
	}

	c.logger.InfoContext(ctx, "deploying with helm",
		slog.String("release", release.Name),
		slog.String("namespace", release.Namespace),
		slog.String("chart", release.Chart),
	)

	return helm.Upgrade(ctx, release)
}

func (c *helmCasting) Melt(ctx context.Context, config installation.Casting, poursPath string, toolers []tooler.Tooler) error {
	helm, err := helmtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := c.identity(config)

	c.logger.InfoContext(ctx, "removing helm release",
		slog.String("release", release.Name),
		slog.String("namespace", release.Namespace),
	)

	return helm.Uninstall(ctx, release)
}

func (c *helmCasting) identity(config installation.Casting) helmtooler.Release {
	return helmtooler.Release{
		Release:   domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Namespace: config.Metadata.Name,
	}
}

func (c *helmCasting) release(config installation.Casting, poursPath string) (helmtooler.Release, error) {
	valuesFile := filepath.Join(poursPath, rootcasting.DeploymentDir, "values.yaml")
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		return helmtooler.Release{}, errors.Newf(errors.TypeNotFound, "values.yaml does not exist at path %s, run 'forge' first", valuesFile)
	}

	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return helmtooler.Release{}, errors.Wrapf(err, errors.TypeInternal, "failed to read values file")
	}

	values := map[string]any{}
	if err := yaml.Unmarshal(valuesBytes, &values); err != nil {
		return helmtooler.Release{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to parse values")
	}

	release := c.identity(config)
	release.Values = values
	release.Chart = installation.HelmChart.Resolve(config.Metadata.Annotations)
	release.Version = installation.HelmChartVersion.Resolve(config.Metadata.Annotations)
	release.Repo = helmtooler.Repo{
		Name: installation.HelmChartRepoName.Resolve(config.Metadata.Annotations),
		URL:  installation.HelmChartRepoURL.Resolve(config.Metadata.Annotations),
	}

	return release, nil
}
