package kuberneteshelmcasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"sigs.k8s.io/yaml"
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

func (c *helmCasting) Cast(ctx context.Context, config installation.Casting, poursPath string) error {

	valuesFile := filepath.Join(poursPath, rootcasting.DeploymentDir, "values.yaml")
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		return errors.Newf(errors.TypeNotFound, "values.yaml does not exist at path %s, run 'forge' first", valuesFile)
	}

	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to read values file")
	}

	vals := map[string]any{}
	if err := yaml.Unmarshal(valuesBytes, &vals); err != nil {
		return errors.Wrapf(err, errors.TypeInvalidInput, "failed to parse values")
	}

	settings := cli.New()
	settings.SetNamespace(config.Metadata.Name)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), config.Metadata.Name, os.Getenv("HELM_DRIVER"), func(format string, v ...any) {
		c.logger.Debug(fmt.Sprintf(format, v...))
	}); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to initialize helm action config")
	}

	chartRef, version, repoURL := chartSource(config)

	c.logger.InfoContext(ctx, "Deploying with Helm",
		slog.String("release", config.Metadata.Name),
		slog.String("chart", chartRef),
		slog.String("repo", repoURL),
		slog.String("namespace", config.Metadata.Name),
	)

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	_, err = histClient.Run(config.Metadata.Name)

	if err != nil {
		install := action.NewInstall(actionConfig)
		install.ReleaseName = config.Metadata.Name
		install.Namespace = config.Metadata.Name
		install.CreateNamespace = true
		install.Version = version
		install.RepoURL = repoURL
		// Readiness belongs to the platform; no other casting waits on it.

		chartPath, err := install.LocateChart(chartRef, settings)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to locate chart")
		}

		chart, err := loader.Load(chartPath)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to load chart")
		}

		c.logger.InfoContext(ctx, "resolved chart", slog.String("chart", chartRef), slog.String("version", chart.Metadata.Version), slog.String("app_version", chart.Metadata.AppVersion))

		if _, err := install.RunWithContext(ctx, chart, vals); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "helm install failed")
		}
	} else {
		upgrade := action.NewUpgrade(actionConfig)
		upgrade.Namespace = config.Metadata.Name
		upgrade.Version = version
		upgrade.RepoURL = repoURL

		chartPath, err := upgrade.LocateChart(chartRef, settings)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to locate chart")
		}

		chart, err := loader.Load(chartPath)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to load chart")
		}

		c.logger.InfoContext(ctx, "resolved chart", slog.String("chart", chartRef), slog.String("version", chart.Metadata.Version), slog.String("app_version", chart.Metadata.AppVersion))

		if _, err := upgrade.RunWithContext(ctx, config.Metadata.Name, chart, vals); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "helm upgrade failed")
		}
	}

	c.logger.InfoContext(ctx, "Helm deployment complete",
		slog.String("release", config.Metadata.Name),
		slog.String("namespace", config.Metadata.Name),
	)
	return nil
}

// The repository applies to a bare chart name only: a reference carrying a slash
// states its own location, and helm would look that string up in the index.
func chartSource(config installation.Casting) (chart, version, repoURL string) {
	chart = installation.HelmChart.Resolve(config.Metadata.Annotations)
	version = installation.HelmChartVersion.Resolve(config.Metadata.Annotations)

	if strings.ContainsRune(chart, '/') {
		return chart, version, ""
	}

	return chart, version, installation.HelmChartRepoURL.Resolve(config.Metadata.Annotations)
}
