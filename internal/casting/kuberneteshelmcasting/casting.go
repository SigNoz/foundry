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

const (
	helmChartRepoUrl  = "https://charts.signoz.io"
	helmChartRepoName = "signoz"
	helmChart         = "signoz/signoz"

	annotationChart      = "foundry.signoz.io/kubernetes-helm-casting-chart"
	annotationRepoURL    = "foundry.signoz.io/kubernetes-helm-casting-repo-url"
	annotationRepoName   = "foundry.signoz.io/kubernetes-helm-casting-repo-name"
	annotationForgeChart = "foundry.signoz.io/kubernetes-helm-casting-forge-chart"
)

var _ rootcasting.Casting = (*helmCasting)(nil)

type helmCasting struct {
	logger  *slog.Logger
	casting *domain.Template
}

func New(logger *slog.Logger) *helmCasting {
	return &helmCasting{
		logger:  logger,
		casting: valuesYAMLTemplate,
	}
}

func (c *helmCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newHelmMoldingEnricher(config), nil
}

func (c *helmCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := valuesYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to execute values yaml template")
	}

	valuesBytes := buf.Bytes()

	valuesMaterial, err := domain.NewYAMLMaterial(valuesBytes, filepath.Join(rootcasting.DeploymentDir, "values.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create values yaml material")
	}

	return []domain.Material{valuesMaterial}, nil
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

	c.logger.InfoContext(ctx, "removing helm release",
		slog.String("release", config.Metadata.Name),
		slog.String("namespace", config.Metadata.Name),
	)

	return helm.Uninstall(ctx, helmtooler.Release{
		Release:   domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Namespace: config.Metadata.Name,
	})
}

// release resolves the forged values and the chart the deploy uses: a local
// chart when the casting forged one, the signoz repo otherwise.
func (c *helmCasting) release(config installation.Casting, poursPath string) (helmtooler.Release, error) {
	valuesFile := filepath.Join(poursPath, rootcasting.DeploymentDir, "values.yaml")
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		return helmtooler.Release{}, errors.Newf(errors.TypeNotFound, "values.yaml does not exist at path %s, run 'forge' first", valuesFile)
	}

	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return helmtooler.Release{}, errors.Wrapf(err, errors.TypeInternal, "failed to read values file")
	}

	vals := map[string]any{}
	if err := yaml.Unmarshal(valuesBytes, &vals); err != nil {
		return helmtooler.Release{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to parse values")
	}

	release := helmtooler.Release{
		Release:   domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Namespace: config.Metadata.Name,
		Values:    vals,
	}

	if c.shouldForgeChart(&config) {
		chartPath := filepath.Join(poursPath, rootcasting.DeploymentDir, "chart", "signoz")
		if _, err := os.Stat(chartPath); os.IsNotExist(err) {
			return helmtooler.Release{}, errors.Newf(errors.TypeNotFound, "local chart not found at %s, run 'forge' first with %s annotation set to 'true'", chartPath, annotationForgeChart)
		}

		release.Chart = chartPath

		return release, nil
	}

	release.Chart = helmChart
	release.Repo = helmtooler.Repo{Name: helmChartRepoName, URL: helmChartRepoUrl}

	if config.Metadata.Annotations != nil {
		if url := config.Metadata.Annotations[annotationRepoURL]; url != "" {
			release.Repo.URL = url
		}

		if chart := config.Metadata.Annotations[annotationChart]; chart != "" {
			release.Chart = chart
		}

		if name := config.Metadata.Annotations[annotationRepoName]; name != "" {
			release.Repo.Name = name
		}
	}

	return release, nil
}

func (c *helmCasting) shouldForgeChart(config *installation.Casting) bool {
	if config.Metadata.Annotations == nil {
		return false
	}

	return config.Metadata.Annotations[annotationForgeChart] == "true"
}
