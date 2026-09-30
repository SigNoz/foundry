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
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"sigs.k8s.io/yaml"
)

const (
	kubeStackChart        = "opentelemetry-kube-stack"
	kubeStackRepoURL      = "https://open-telemetry.github.io/opentelemetry-helm-charts"
	kubeStackChartVersion = "0.13.0"
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
	var tmpl *domain.Template

	switch ctrl := controller(config); ctrl {
	case collectionagent.CollectorControllerDefault:
		tmpl = valuesYAMLTemplate
	case collectionagent.CollectorControllerOpenTelemetryOperator:
		tmpl = kubeStackValuesYAMLTemplate
	default:
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "collector controller %q is not supported, state %q or %q", ctrl, collectionagent.CollectorControllerDefault, collectionagent.CollectorControllerOpenTelemetryOperator)
	}

	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, templateDataFor(config)); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", tmpl.Name())
	}

	// A values file per kind, so a casting file of several agents pours one per document.
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
	ns := namespace(config)

	settings := cli.New()
	settings.SetNamespace(ns)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), ns, os.Getenv("HELM_DRIVER"), func(format string, v ...any) {
		c.logger.DebugContext(ctx, fmt.Sprintf(format, v...))
	}); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to initialize helm action config")
	}

	chartRef, version, repoURL := chartSource(config)

	c.logger.InfoContext(ctx, "Deploying with Helm",
		slog.String("release", release),
		slog.String("chart", chartRef),
		slog.String("repo", repoURL),
		slog.String("namespace", ns),
	)

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1

	if _, err := histClient.Run(release); err != nil {
		install := action.NewInstall(actionConfig)
		install.ReleaseName = release
		install.Namespace = ns
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
		upgrade.Namespace = ns
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
		slog.String("namespace", ns),
	)

	return nil
}

// The embedded casting keeps $.Spec and $.Metadata reachable from the templates.
type templateData struct {
	collectionagent.Casting

	Namespace string
}

func templateDataFor(config collectionagent.Casting) templateData {
	return templateData{Casting: config, Namespace: namespace(config)}
}

// The annotation's default cannot name metadata.name, so the fallback lives here.
func namespace(config collectionagent.Casting) string {
	ns := collectionagent.KubernetesNamespace.Resolve(config.Metadata.Annotations)
	if ns == "" {
		ns = config.Metadata.Name
	}

	return ns
}

// The kind keeps a file's agent and deployment two releases in one namespace.
func releaseName(config collectionagent.Casting) string {
	return fmt.Sprintf("%s-collector-%s", config.Metadata.Name, config.Spec.Collector.Kind)
}

func controller(config collectionagent.Casting) string {
	return collectionagent.KubernetesCollectorController.Resolve(config.Metadata.Annotations)
}

// The repository applies to a bare chart name only: helm looks a slashed reference up in the index.
// Under the operator an unstated chart, repository or version names the kube-stack chart; a stated one wins.
func chartSource(config collectionagent.Casting) (chart, version, repoURL string) {
	annotations := config.Metadata.Annotations

	chart = collectionagent.HelmChart.Resolve(annotations)
	version = collectionagent.HelmChartVersion.Resolve(annotations)
	repoURL = collectionagent.HelmChartRepoURL.Resolve(annotations)

	if controller(config) == collectionagent.CollectorControllerOpenTelemetryOperator {
		if annotations[collectionagent.HelmChart.Key] == "" {
			chart = kubeStackChart
		}

		if annotations[collectionagent.HelmChartVersion.Key] == "" {
			version = kubeStackChartVersion
		}

		if annotations[collectionagent.HelmChartRepoURL.Key] == "" {
			repoURL = kubeStackRepoURL
		}
	}

	// Helm has no "latest" token: only an empty version means the repository's newest chart.
	if version == "latest" {
		version = ""
	}

	if strings.ContainsRune(chart, '/') {
		return chart, version, ""
	}

	return chart, version, repoURL
}
