package kuberneteshelmcasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

const (
	helmChartRepo     = "https://charts.signoz.io"
	helmRepoName      = "signoz"
	helmChartName     = "signoz/signoz"
	helmDeployTimeout = 10 * time.Minute

	defaultHelmNamespace = "platform"

	annotationNamespace  = "foundry.signoz.io/kubernetes-helm-casting-namespace"
	annotationChart      = "foundry.signoz.io/kubernetes-helm-casting-chart"
	annotationRepoURL    = "foundry.signoz.io/kubernetes-helm-casting-repo-url"
	annotationForgeChart = "foundry.signoz.io/kubernetes-helm-casting-forge-chart"
)

var _ rootcasting.Casting = (*helmCasting)(nil)

type helmCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *helmCasting {
	return &helmCasting{logger: logger}
}

func (c *helmCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newHelmMoldingEnricher(config), nil
}

func (c *helmCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := valuesYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute values yaml template: %w", err)
	}

	valuesMaterial, err := types.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "values.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to create values yaml material: %w", err)
	}

	return []types.Material{valuesMaterial}, nil
}

func (c *helmCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	valuesFile := filepath.Join(poursPath, rootcasting.DeploymentDir, "values.yaml")
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		return fmt.Errorf("values.yaml does not exist at path %s, run 'forge' first", valuesFile)
	}

	namespace := defaultHelmNamespace
	if config.Metadata.Annotations != nil {
		if ns := config.Metadata.Annotations[annotationNamespace]; ns != "" {
			namespace = ns
		}
	}

	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "", func(format string, v ...interface{}) {
		c.logger.Info(fmt.Sprintf(format, v...))
	}); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return fmt.Errorf("failed to read values file: %w", err)
	}

	vals, err := parseValues(valuesBytes)
	if err != nil {
		return fmt.Errorf("failed to parse values: %w", err)
	}

	var chartRef string
	if c.shouldForgeChart(&config) {
		chartRef = filepath.Join(poursPath, rootcasting.DeploymentDir, "chart", "signoz")
		if _, err := os.Stat(chartRef); os.IsNotExist(err) {
			return fmt.Errorf("local chart not found at %s, run 'forge' first with %s annotation set to 'true'", chartRef, annotationForgeChart)
		}
		c.logger.InfoContext(ctx, "Installing from local chart", slog.String("path", chartRef))
	} else {
		repoURL := helmChartRepo
		if config.Metadata.Annotations != nil {
			if u := config.Metadata.Annotations[annotationRepoURL]; u != "" {
				repoURL = u
			}
		}

		chartRef = helmChartName
		if config.Metadata.Annotations != nil {
			if ch := config.Metadata.Annotations[annotationChart]; ch != "" {
				chartRef = ch
			}
		}

		c.logger.InfoContext(ctx, "Adding Helm repo", slog.String("name", helmRepoName), slog.String("url", repoURL))
		if err := addHelmRepo(settings, helmRepoName, repoURL); err != nil {
			return fmt.Errorf("failed to add helm repo: %w", err)
		}
	}

	c.logger.InfoContext(ctx, "Deploying with Helm",
		slog.String("release", config.Metadata.Name),
		slog.String("chart", chartRef),
		slog.String("namespace", namespace),
	)

	histClient := action.NewHistory(actionConfig)
	histClient.Max = 1
	_, err = histClient.Run(config.Metadata.Name)
	releaseExists := err == nil

	if !releaseExists {
		install := action.NewInstall(actionConfig)
		install.ReleaseName = config.Metadata.Name
		install.Namespace = namespace
		install.CreateNamespace = true
		install.Wait = true
		install.Timeout = helmDeployTimeout

		chartPath, err := install.LocateChart(chartRef, settings)
		if err != nil {
			return fmt.Errorf("failed to locate chart: %w", err)
		}

		chart, err := loader.Load(chartPath)
		if err != nil {
			return fmt.Errorf("failed to load chart: %w", err)
		}

		if _, err := install.RunWithContext(ctx, chart, vals); err != nil {
			return fmt.Errorf("helm install failed: %w", err)
		}
	} else {
		upgrade := action.NewUpgrade(actionConfig)
		upgrade.Namespace = namespace
		upgrade.Wait = true
		upgrade.Timeout = helmDeployTimeout

		chartPath, err := upgrade.LocateChart(chartRef, settings)
		if err != nil {
			return fmt.Errorf("failed to locate chart: %w", err)
		}

		chart, err := loader.Load(chartPath)
		if err != nil {
			return fmt.Errorf("failed to load chart: %w", err)
		}

		if _, err := upgrade.RunWithContext(ctx, config.Metadata.Name, chart, vals); err != nil {
			return fmt.Errorf("helm upgrade failed: %w", err)
		}
	}

	c.logger.InfoContext(ctx, "Helm deployment complete",
		slog.String("release", config.Metadata.Name),
		slog.String("namespace", namespace),
	)
	return nil
}

func (c *helmCasting) NeedsMoldings() bool {
	return false
}

func (c *helmCasting) shouldForgeChart(config *v1alpha1.Casting) bool {
	if config.Metadata.Annotations == nil {
		return false
	}
	return config.Metadata.Annotations[annotationForgeChart] == "true"
}

func addHelmRepo(settings *cli.EnvSettings, name, url string) error {
	repoFile := settings.RepositoryConfig
	repoEntry := &repo.Entry{
		Name: name,
		URL:  url,
	}

	r, err := repo.NewChartRepository(repoEntry, getter.All(settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	r.CachePath = settings.RepositoryCache
	if _, err := r.DownloadIndexFile(); err != nil {
		return fmt.Errorf("failed to download repo index: %w", err)
	}

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		f = repo.NewFile()
	}

	f.Update(repoEntry)
	return f.WriteFile(repoFile, 0644)
}

func parseValues(data []byte) (map[string]any, error) {
	vals := map[string]any{}
	if err := yaml.Unmarshal(data, &vals); err != nil {
		return nil, err
	}
	return vals, nil
}
