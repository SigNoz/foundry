package kuberneteshelmcasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"sigs.k8s.io/yaml"
)

type kubernetesHelmCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *kubernetesHelmCasting {
	return &kubernetesHelmCasting{logger: logger}
}

func (c *kubernetesHelmCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newKubernetesHelmMoldingEnricher(), nil
}

func (c *kubernetesHelmCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	buf := bytes.NewBuffer(nil)
	if err := valuesYAMLTemplate.Execute(buf, config); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", valuesYAMLTemplate.Name())
	}

	// One values file per kind, so a casting file holding several collection
	// agents pours a release's values per document.
	p.AddYAML(buf.Bytes(), filepath.Dir(config.Spec.Collector.Kind.ConfigKey()), "values.yaml")

	return nil
}

func (c *kubernetesHelmCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	valuesFile := filepath.Join(outputPath, p.Dir(), filepath.Dir(config.Spec.Collector.Kind.ConfigKey()), "values.yaml")
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "values.yaml does not exist at path %s, run 'forge' first", valuesFile)
	}

	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to read values file")
	}

	vals := map[string]any{}
	if err := yaml.Unmarshal(valuesBytes, &vals); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to parse values")
	}

	release := releaseName(config)

	settings := cli.New()
	settings.SetNamespace(config.Metadata.Name)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), config.Metadata.Name, os.Getenv("HELM_DRIVER"), func(format string, v ...any) {
		c.logger.DebugContext(ctx, fmt.Sprintf(format, v...))
	}); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to initialize helm action config")
	}

	chartRef, version, repoURL := chartSource(config)

	c.logger.InfoContext(ctx, "Deploying with Helm",
		slog.String("release", release),
		slog.String("chart", chartRef),
		slog.String("repo", repoURL),
		slog.String("namespace", config.Metadata.Name),
	)

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1

	if _, err := histClient.Run(release); err != nil {
		install := action.NewInstall(actionConfig)
		install.ReleaseName = release
		install.Namespace = config.Metadata.Name
		install.CreateNamespace = true
		install.Version = version
		install.RepoURL = repoURL
		// Readiness belongs to the platform; no other casting waits on it.

		chartPath, err := install.LocateChart(chartRef, settings)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to locate chart")
		}

		chart, err := loader.Load(chartPath)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to load chart")
		}

		c.logger.InfoContext(ctx, "resolved chart", slog.String("chart", chartRef), slog.String("version", chart.Metadata.Version), slog.String("app_version", chart.Metadata.AppVersion))

		if _, err := install.RunWithContext(ctx, chart, vals); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "helm install failed")
		}
	} else {
		upgrade := action.NewUpgrade(actionConfig)
		upgrade.Namespace = config.Metadata.Name
		upgrade.Version = version
		upgrade.RepoURL = repoURL

		chartPath, err := upgrade.LocateChart(chartRef, settings)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to locate chart")
		}

		chart, err := loader.Load(chartPath)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to load chart")
		}

		c.logger.InfoContext(ctx, "resolved chart", slog.String("chart", chartRef), slog.String("version", chart.Metadata.Version), slog.String("app_version", chart.Metadata.AppVersion))

		if _, err := upgrade.RunWithContext(ctx, release, chart, vals); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "helm upgrade failed")
		}
	}

	c.logger.InfoContext(ctx, "Helm deployment complete",
		slog.String("release", release),
		slog.String("namespace", config.Metadata.Name),
	)

	return nil
}

// The release name carries the collector kind so the agent and the deployment
// of one casting file live as two releases in the same namespace.
func releaseName(config collectionagent.Casting) string {
	return fmt.Sprintf("%s-collector-%s", config.Metadata.Name, config.Spec.Collector.Kind)
}

// The repository applies to a bare chart name only: a reference carrying a slash
// states its own location, and helm would look that string up in the index.
func chartSource(config collectionagent.Casting) (chart, version, repoURL string) {
	chart = collectionagent.HelmChart.Resolve(config.Metadata.Annotations)
	version = collectionagent.HelmChartVersion.Resolve(config.Metadata.Annotations)

	// Helm has no "latest" token: it parses a stated version as a semver
	// constraint, and only an empty one means the repository's newest chart.
	if version == "latest" {
		version = ""
	}

	if strings.ContainsRune(chart, '/') {
		return chart, version, ""
	}

	return chart, version, collectionagent.HelmChartRepoURL.Resolve(config.Metadata.Annotations)
}
