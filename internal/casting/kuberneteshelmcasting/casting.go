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
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

const (
	helmChartRepoUrl  = "https://charts.signoz.io"
	helmChartRepoName = "signoz"
	helmChart         = "signoz/signoz"
	helmDeployTimeout = 10 * time.Minute

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

func (c *helmCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newHelmMoldingEnricher(config), nil
}

func (c *helmCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]domain.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := valuesYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute values yaml template: %w", err)
	}

	valuesBytes := buf.Bytes()

	valuesMaterial, err := domain.NewYAMLMaterial(valuesBytes, filepath.Join(rootcasting.DeploymentDir, "values.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to create values yaml material: %w", err)
	}

	return []domain.Material{valuesMaterial}, nil
}

func (c *helmCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {

	valuesFile := filepath.Join(poursPath, rootcasting.DeploymentDir, "values.yaml")
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		return fmt.Errorf("values.yaml does not exist at path %s, run 'forge' first", valuesFile)
	}

	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return fmt.Errorf("failed to read values file: %w", err)
	}

	vals := map[string]any{}
	if err := yaml.Unmarshal(valuesBytes, &vals); err != nil {
		return fmt.Errorf("failed to parse values: %w", err)
	}

	settings := cli.New()
	settings.SetNamespace(config.Metadata.Name)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), config.Metadata.Name, os.Getenv("HELM_DRIVER"), func(format string, v ...any) {
		c.logger.Debug(fmt.Sprintf(format, v...))
	}); err != nil {
		return fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	var chartRef string
	if c.shouldForgeChart(&config) {
		chartRef = filepath.Join(poursPath, rootcasting.DeploymentDir, "chart", "signoz")
		if _, err := os.Stat(chartRef); os.IsNotExist(err) {
			return fmt.Errorf("local chart not found at %s, run 'forge' first with %s annotation set to 'true'", chartRef, annotationForgeChart)
		}
		c.logger.InfoContext(ctx, "Installing from local chart", slog.String("path", chartRef))
	} else {
		repoURL := helmChartRepoUrl
		if config.Metadata.Annotations != nil {
			if u := config.Metadata.Annotations[annotationRepoURL]; u != "" {
				repoURL = u
			}
		}

		chartRef = helmChart
		if config.Metadata.Annotations != nil {
			if ch := config.Metadata.Annotations[annotationChart]; ch != "" {
				chartRef = ch
			}
		}

		repoName := helmChartRepoName
		if config.Metadata.Annotations != nil {
			if ch := config.Metadata.Annotations[annotationRepoName]; ch != "" {
				chartRef = ch
			}
		}

		c.logger.InfoContext(ctx, "Adding Helm repo", slog.String("name", repoName), slog.String("url", repoURL), slog.String("chart", chartRef))
		if err := addHelmRepo(settings, repoName, repoURL); err != nil {
			return fmt.Errorf("failed to add helm repo: %w", err)
		}
	}

	c.logger.InfoContext(ctx, "Deploying with Helm",
		slog.String("release", config.Metadata.Name),
		slog.String("chart", chartRef),
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
		upgrade.Namespace = config.Metadata.Name
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
		slog.String("namespace", config.Metadata.Name),
	)
	return nil
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
