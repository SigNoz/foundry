package kuberneteshelmcasting

import (
	"bytes"
	"context"
	"encoding/json"
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
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

const (
	helmChartRepoUrl     = "https://charts.signoz.io"
	helmChartRepoName      = "signoz"
	helmChart    = "signoz/signoz"
	helmDeployTimeout = 10 * time.Minute

	annotationChart      = "foundry.signoz.io/kubernetes-helm-casting-chart"
	annotationRepoURL    = "foundry.signoz.io/kubernetes-helm-casting-repo-url"
	annotationRepoName   = "foundry.signoz.io/kubernetes-helm-casting-repo-name"
	annotationForgeChart = "foundry.signoz.io/kubernetes-helm-casting-forge-chart"
)

var _ rootcasting.Casting = (*helmCasting)(nil)

// HelmKnobs defines the supported knobs for the helm casting.
// Knobs map directly to SigNoz Helm chart values.
type HelmKnobs struct {
	// Resources defines CPU and memory requests/limits.
	Resources map[string]any `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Tolerations defines pod tolerations for scheduling.
	Tolerations []map[string]any `json:"tolerations,omitempty" yaml:"tolerations,omitempty"`

	// NodeSelector defines node selection constraints.
	NodeSelector map[string]string `json:"nodeSelector,omitempty" yaml:"nodeSelector,omitempty"`

	// Affinity defines pod affinity and anti-affinity rules.
	Affinity map[string]any `json:"affinity,omitempty" yaml:"affinity,omitempty"`

	// TopologySpreadConstraints defines how pods are spread across topology domains.
	TopologySpreadConstraints []map[string]any `json:"topologySpreadConstraints,omitempty" yaml:"topologySpreadConstraints,omitempty"`

	// PodSecurityContext defines the pod-level security context.
	PodSecurityContext map[string]any `json:"podSecurityContext,omitempty" yaml:"podSecurityContext,omitempty"`

	// SecurityContext defines the container-level security context.
	SecurityContext map[string]any `json:"securityContext,omitempty" yaml:"securityContext,omitempty"`

	// ImagePullSecrets lists secret names for pulling container images.
	ImagePullSecrets []map[string]any `json:"imagePullSecrets,omitempty" yaml:"imagePullSecrets,omitempty"`

	// PodAnnotations defines annotations to add to the pod template.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty" yaml:"podAnnotations,omitempty"`

	// PodLabels defines extra labels to add to the pod template.
	PodLabels map[string]string `json:"podLabels,omitempty" yaml:"podLabels,omitempty"`

	// Persistence defines storage configuration (size, storageClass).
	Persistence map[string]any `json:"persistence,omitempty" yaml:"persistence,omitempty"`

	// Service defines service configuration (type, annotations, labels).
	Service map[string]any `json:"service,omitempty" yaml:"service,omitempty"`
}

type helmCasting struct {
	logger  *slog.Logger
	casting *types.Template
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

func (c *helmCasting) Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	if err := c.validateKnobs(config); err != nil {
		return nil, fmt.Errorf("invalid knobs: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	err := valuesYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute values yaml template: %w", err)
	}

	valuesBytes := buf.Bytes()

	// Merge patches into base values. Each patch is a partial values YAML
	// that overrides base values via CoalesceTables (patch wins, base fills gaps).
	if len(config.Patches) > 0 {
		baseVals := map[string]any{}
		if err := yaml.Unmarshal(valuesBytes, &baseVals); err != nil {
			return nil, fmt.Errorf("failed to parse base values: %w", err)
		}

		for _, patch := range config.Patches {
			if patch.Path == "" {
				continue
			}

			patchBytes, err := os.ReadFile(patch.Path)
			if err != nil {
				return nil, fmt.Errorf("failed to read patch file %q: %w", patch.Path, err)
			}

			patchVals := map[string]any{}
			if err := yaml.Unmarshal(patchBytes, &patchVals); err != nil {
				return nil, fmt.Errorf("failed to parse patch file %q: %w", patch.Path, err)
			}

			baseVals = chartutil.CoalesceTables(patchVals, baseVals)
		}

		valuesBytes, err = yaml.Marshal(baseVals)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal merged values: %w", err)
		}
	}

	valuesMaterial, err := types.NewYAMLMaterial(valuesBytes, filepath.Join(rootcasting.DeploymentDir, "values.yaml"))
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
	if err := actionConfig.Init(settings.RESTClientGetter(), config.Metadata.Name, os.Getenv("HELM_DRIVER"), c.logger.Debug); err != nil {
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

// validateKnobs parses each component's knobs into HelmKnobs to catch
// type mismatches and unknown keys before templates run.
func (c *helmCasting) validateKnobs(cfg v1alpha1.Casting) error {
	components := map[string]any{
		"signoz":          cfg.Spec.Signoz.Spec.Config.Knobs,
		"ingester":        cfg.Spec.Ingester.Spec.Config.Knobs,
		"telemetrystore":  cfg.Spec.TelemetryStore.Spec.Config.Knobs,
		"telemetrykeeper": cfg.Spec.TelemetryKeeper.Spec.Config.Knobs,
		"metastore":       cfg.Spec.MetaStore.Spec.Config.Knobs,
	}

	for component, knobs := range components {
		if knobs == nil {
			continue
		}

		data, err := json.Marshal(knobs)
		if err != nil {
			return fmt.Errorf("component %s: failed to marshal knobs: %w", component, err)
		}

		var k HelmKnobs
		if err := json.Unmarshal(data, &k); err != nil {
			return fmt.Errorf("component %s: %w", component, err)
		}
	}

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
