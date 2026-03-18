package kuberneteskustomizecasting

import (
	"bytes"
	"context"
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
			deploymentNamespace,
			deploymentKustomization,
		},
	}
}

func (c *kustomizeCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newKustomizeMoldingEnricher(config)
}

func (c *kustomizeCasting) Forge(ctx context.Context, cfg v1alpha1.Casting, poursPath string) ([]types.Material, error) {
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

func getOverrideMaterials(config *v1alpha1.Casting) ([]types.Material, error) {
	var materials []types.Material

	storeBuf := bytes.NewBuffer(nil)
	if err := telemetryStoreOverrideTemplate.Execute(storeBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute store override template: %w", err)
	}
	storeMaterial, err := types.NewYAMLMaterial(storeBuf.Bytes(), "store_overrides.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create store override material: %w", err)
	}
	materials = append(materials, storeMaterial)

	return materials, nil
}

func getServiceMaterials(config *v1alpha1.Casting) ([]types.Material, error) {
	var materials []types.Material

	telemetryStoreInstallationBuf := bytes.NewBuffer(nil)
	if err := clickhouseInstanceInstallation.Execute(telemetryStoreInstallationBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute store installation template: %w", err)
	}
	telemetryStoreInstallationMaterial, err := types.NewYAMLMaterial(telemetryStoreInstallationBuf.Bytes(), "clickhouseInstallation.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create keeper override material: %w", err)
	}
	materials = append(materials, telemetryStoreInstallationMaterial)

	metaStoreServiceBuf := bytes.NewBuffer(nil)
	if err := metastoreService.Execute(metaStoreServiceBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute store installation template: %w", err)
	}
	metaStoreServiceMaterial, err := types.NewYAMLMaterial(metaStoreServiceBuf.Bytes(), "metastoreServie.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create metastore service material: %w", err)
	}
	materials = append(materials, metaStoreServiceMaterial)
	return materials, nil
}
