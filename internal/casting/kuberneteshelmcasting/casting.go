package kuberneteshelmcasting

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
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
	templates := []*domain.Template{valuesYAMLTemplate}
	if config.Spec.MCP.Spec.IsEnabled() {
		templates = append(templates, mcpValuesYAMLTemplate)
	}

	var materials []domain.Material
	for _, tmpl := range templates {
		name := tmpl.Name()
		path := filepath.Join(rootcasting.DeploymentDir, strings.TrimSuffix(name, filepath.Ext(name)))

		material, err := tmpl.Render(config, path)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to render %q", name)
		}

		materials = append(materials, material)
	}

	return materials, nil
}

func (c *helmCasting) Cast(ctx context.Context, config installation.Casting, poursPath string) error {
	ns := namespace(config)

	for _, kind := range []v1alpha1.MoldingKind{v1alpha1.MoldingKindSignoz, v1alpha1.MoldingKindMCP} {
		var (
			tmpl                             *domain.Template
			release, chart, version, repoURL string
		)

		switch kind {
		case v1alpha1.MoldingKindSignoz:
			if !config.Spec.Signoz.Spec.IsEnabled() {
				return errors.Newf(errors.TypeUnsupported, "installation '%s/%s' cannot turn off signoz: the helm chart does not allow it, raise an issue at https://github.com/signoz/foundry/issues", config.Spec.Deployment.Mode, config.Spec.Deployment.Flavor)
			}

			if !config.Spec.Ingester.Spec.IsEnabled() {
				return errors.Newf(errors.TypeUnsupported, "installation '%s/%s' cannot turn off the ingester: the helm chart does not allow it, raise an issue at https://github.com/signoz/foundry/issues", config.Spec.Deployment.Mode, config.Spec.Deployment.Flavor)
			}

			tmpl = valuesYAMLTemplate
			release = config.Metadata.Name
			chart = installation.HelmChart.Resolve(config.Metadata.Annotations)
			version = installation.HelmChartVersion.Resolve(config.Metadata.Annotations)
			repoURL = installation.HelmChartRepoURL.Resolve(config.Metadata.Annotations)

		case v1alpha1.MoldingKindMCP:
			if !config.Spec.MCP.Spec.IsEnabled() {
				continue
			}

			tmpl = mcpValuesYAMLTemplate
			release = config.Metadata.Name + "-" + kind.String() + "-server"
			chart = installation.HelmMCPChart.Resolve(config.Metadata.Annotations)
			version = installation.HelmMCPChartVersion.Resolve(config.Metadata.Annotations)
			repoURL = installation.HelmMCPChartRepoURL.Resolve(config.Metadata.Annotations)
		}

		// Helm has no "latest" token: only an empty version means the repository's newest chart.
		if version == "latest" {
			version = ""
		}

		// A chart reference carrying a slash states its own location; the repository applies to a bare name only.
		if strings.ContainsRune(chart, '/') {
			repoURL = ""
		}

		name := tmpl.Name()
		valuesFile := filepath.Join(poursPath, rootcasting.DeploymentDir, strings.TrimSuffix(name, filepath.Ext(name)))

		if err := c.installOrUpgrade(ctx, release, ns, chart, version, repoURL, valuesFile); err != nil {
			return err
		}
	}

	return nil
}

func (c *helmCasting) installOrUpgrade(ctx context.Context, release, ns, chartRef, version, repoURL, valuesFile string) error {
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		return errors.Newf(errors.TypeNotFound, "%q does not exist, run 'forge' first", valuesFile)
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
	settings.SetNamespace(ns)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), ns, os.Getenv("HELM_DRIVER"), func(format string, v ...any) {
		c.logger.Debug(fmt.Sprintf(format, v...))
	}); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to initialize helm action config")
	}

	c.logger.InfoContext(ctx, "Deploying with Helm",
		slog.String("release", release),
		slog.String("chart", chartRef),
		slog.String("repo", repoURL),
		slog.String("namespace", ns),
	)

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	_, err = histClient.Run(release)

	if err != nil {
		install := action.NewInstall(actionConfig)
		install.ReleaseName = release
		install.Namespace = ns
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
		upgrade.Namespace = ns
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

		if _, err := upgrade.RunWithContext(ctx, release, chart, vals); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "helm upgrade failed")
		}
	}

	c.logger.InfoContext(ctx, "Helm deployment complete",
		slog.String("release", release),
		slog.String("namespace", ns),
	)
	return nil
}

// The annotation's default cannot name metadata.name, so the fallback lives here.
func namespace(config installation.Casting) string {
	ns := installation.KubernetesNamespace.Resolve(config.Metadata.Annotations)
	if ns == "" {
		ns = config.Metadata.Name
	}

	return ns
}
