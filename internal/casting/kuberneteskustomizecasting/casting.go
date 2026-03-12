package kuberneteskustomizecasting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ rootcasting.Casting = (*kustomizeCasting)(nil)

type kustomizeCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

// KustomizeKnobs defines the supported knobs for the kustomize casting.
type KustomizeKnobs struct {
	// Resources defines CPU and memory requests/limits for the container.
	Resources map[string]any `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Tolerations defines pod tolerations for scheduling.
	Tolerations []map[string]any `json:"tolerations,omitempty" yaml:"tolerations,omitempty"`

	// NodeSelector defines node selection constraints.
	NodeSelector map[string]string `json:"nodeSelector,omitempty" yaml:"nodeSelector,omitempty"`

	// Affinity defines pod affinity and anti-affinity rules.
	Affinity map[string]any `json:"affinity,omitempty" yaml:"affinity,omitempty"`

	// TopologySpreadConstraints defines how pods are spread across topology domains.
	TopologySpreadConstraints []map[string]any `json:"topologySpreadConstraints,omitempty" yaml:"topologySpreadConstraints,omitempty"`

	// PodSecurityContext defines the pod-level security context (e.g. runAsUser, fsGroup).
	PodSecurityContext map[string]any `json:"podSecurityContext,omitempty" yaml:"podSecurityContext,omitempty"`

	// SecurityContext defines the container-level security context (e.g. runAsNonRoot, readOnlyRootFilesystem).
	SecurityContext map[string]any `json:"securityContext,omitempty" yaml:"securityContext,omitempty"`

	// ImagePullSecrets lists secret names for pulling container images.
	ImagePullSecrets []map[string]any `json:"imagePullSecrets,omitempty" yaml:"imagePullSecrets,omitempty"`

	// MinReadySeconds is the minimum seconds a pod should be ready before considered available (Deployment/StatefulSet).
	MinReadySeconds *int `json:"minReadySeconds,omitempty" yaml:"minReadySeconds,omitempty"`

	// StorageSize defines the PVC storage request size (e.g. "10Gi").
	StorageSize string `json:"storageSize,omitempty" yaml:"storageSize,omitempty"`

	// StorageClass defines the PVC storage class name.
	StorageClass string `json:"storageClass,omitempty" yaml:"storageClass,omitempty"`

	// ServiceType defines the Kubernetes service type (e.g. "ClusterIP", "LoadBalancer").
	ServiceType string `json:"serviceType,omitempty" yaml:"serviceType,omitempty"`

	// ServiceAnnotations defines annotations to add to the service.
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty" yaml:"serviceAnnotations,omitempty"`

	// ServiceLabels defines labels to add to the service.
	ServiceLabels map[string]string `json:"serviceLabels,omitempty" yaml:"serviceLabels,omitempty"`

	// PodAnnotations defines annotations to add to the pod template.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty" yaml:"podAnnotations,omitempty"`

	// PodLabels defines extra labels to add to the pod template (podExtraLabels in Kubedeploy, podLabels in Altinity).
	PodLabels map[string]string `json:"podLabels,omitempty" yaml:"podLabels,omitempty"`
}

func New(logger *slog.Logger) *kustomizeCasting {
	return &kustomizeCasting{
		logger: logger,
		castings: []*types.Template{
			clickhouseOperatorClusterrole,
			clickhouseOperatorClusterrolebinding,
			clickhouseOperatorConfigmap,
			clickhouseOperatorDeployment,
			clickhouseOperatorService,
			clickhouseOperatorServiceaccount,
			clickhouseInstanceInstallation,
			clickhouseInstanceConfigmap,
			clickhouseKeeperInstallation,
			signozService,
			signozServiceaccount,
			signozStatefulset,
			ingesterConfigmap,
			ingesterDeployment,
			ingesterService,
			ingesterServiceaccount,
			metastoreService,
			metastoreServiceaccount,
			metastoreStatefulset,
			telemetrystoreMigratorJob,
			clickhouseOperatorKustomization,
			clickhouseInstallationKustomization,
			clickhouseKeeperKustomization,
			signozKustomization,
			ingesterKustomization,
			metastoreKustomization,
			telemetrystoreMigratorKustomization,
			deploymentKustomization,
		},
	}
}

func (c *kustomizeCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newKustomizeMoldingEnricher(config), nil
}

func (c *kustomizeCasting) Forge(ctx context.Context, cfg v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	if err := c.validateKnobs(cfg); err != nil {
		return nil, fmt.Errorf("invalid knobs: %w", err)
	}

	if err := c.resolvePatchPaths(&cfg, poursPath); err != nil {
		return nil, fmt.Errorf("failed to resolve patch paths: %w", err)
	}

	var materials []types.Material
	for _, tmpl := range c.castings {
		m, err := c.forgeCasting(tmpl, &cfg, poursPath)
		if err != nil {
			return nil, fmt.Errorf("failed to forge: %w", err)
		}
		materials = append(materials, m...)
	}
	return materials, nil
}

const clickhouseOperatorVersion = "0.25.3"

var clickhouseCRDs = []string{
	"clickhouseinstallations.clickhouse.altinity.com.crd.yaml",
	"clickhouseinstallationtemplates.clickhouse.altinity.com.crd.yaml",
	"clickhouseoperatorconfigurations.clickhouse.altinity.com.crd.yaml",
	"clickhousekeeperinstallations.clickhouse-keeper.altinity.com.crd.yaml",
}

func (c *kustomizeCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Applying kustomize manifests")

	kustomizeDir := filepath.Join(poursPath, rootcasting.DeploymentDir)
	if _, err := os.Stat(filepath.Join(kustomizeDir, "kustomization.yaml")); os.IsNotExist(err) {
		return fmt.Errorf("kustomization.yaml does not exist at path: %s, run 'forge' first", kustomizeDir)
	}

	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := c.applyCRDs(runctx); err != nil {
		return fmt.Errorf("failed to apply CRDs: %w", err)
	}

	cmd := exec.CommandContext(runctx, "kubectl", "apply", "-k", kustomizeDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	c.logger.DebugContext(runctx, "Running command",
		slog.String("command", fmt.Sprintf("kubectl apply -k %s", kustomizeDir)))

	if err := cmd.Run(); err != nil {
		c.logger.ErrorContext(runctx, "kubectl apply failed", slog.String("error", err.Error()))
		return fmt.Errorf("kubectl apply -k failed: %w", err)
	}

	c.logger.InfoContext(runctx, "Kustomize manifests applied successfully")
	return nil
}

func (c *kustomizeCasting) applyCRDs(ctx context.Context) error {
	c.logger.InfoContext(ctx, "Applying ClickHouse CRDs",
		slog.String("version", clickhouseOperatorVersion))

	for _, crd := range clickhouseCRDs {
		url := fmt.Sprintf(
			"https://raw.githubusercontent.com/Altinity/clickhouse-operator/%s/deploy/operatorhub/%s/%s",
			clickhouseOperatorVersion, clickhouseOperatorVersion, crd,
		)

		cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", url)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		c.logger.DebugContext(ctx, "Applying CRD", slog.String("url", url))

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to apply CRD %s: %w", crd, err)
		}
	}

	return nil
}

// validateKnobs parses each component's knobs into KustomizeKnobs to catch
// type mismatches and unknown keys before templates run.
func (c *kustomizeCasting) validateKnobs(cfg v1alpha1.Casting) error {
	components := map[string]map[string]any{
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

		var k KustomizeKnobs
		if err := json.Unmarshal(data, &k); err != nil {
			return fmt.Errorf("component %s: %w", component, err)
		}
	}

	return nil
}

// resolvePatchPaths converts patch file paths from project-root-relative to
// kustomization-directory-relative so kustomize can find them without copying.
func (c *kustomizeCasting) resolvePatchPaths(cfg *v1alpha1.Casting, poursPath string) error {
	if len(cfg.Patches) == 0 {
		return nil
	}

	kustomizeDir := filepath.Join(poursPath, rootcasting.DeploymentDir)
	absKustomizeDir, err := filepath.Abs(kustomizeDir)
	if err != nil {
		return fmt.Errorf("failed to resolve kustomize directory: %w", err)
	}

	for i, patch := range cfg.Patches {
		if patch.Path == "" {
			continue
		}

		absPatchPath, err := filepath.Abs(patch.Path)
		if err != nil {
			return fmt.Errorf("failed to resolve patch path %q: %w", patch.Path, err)
		}

		if _, err := os.Stat(absPatchPath); err != nil {
			return fmt.Errorf("patch file %q does not exist: %w", patch.Path, err)
		}

		relPath, err := filepath.Rel(absKustomizeDir, absPatchPath)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %q: %w", patch.Path, err)
		}

		cfg.Patches[i].Path = relPath
	}

	return nil
}

func (c *kustomizeCasting) forgeCasting(tmpl *types.Template, cfg *v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	templatePath := tmpl.GetPath()
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", templatePath, err)
	}
	relPath := strings.TrimPrefix(templatePath, "templates/")
	relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))
	path := filepath.Join(rootcasting.DeploymentDir, relPath)
	material, err := types.NewYAMLMaterial(buf.Bytes(), path)
	if err != nil {
		return nil, fmt.Errorf("create material %s: %w", templatePath, err)
	}
	return []types.Material{material}, nil
}
