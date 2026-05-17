package kuberneteskustomizecasting

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ rootcasting.Casting = (*kustomizeCasting)(nil)

type kustomizeCasting struct {
	logger   *slog.Logger
	castings []*domain.Template
}

func New(logger *slog.Logger) *kustomizeCasting {
	return &kustomizeCasting{
		logger: logger,
		castings: []*domain.Template{
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

func (c *kustomizeCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newKustomizeMoldingEnricher(config)
}

func (c *kustomizeCasting) Forge(ctx context.Context, cfg installation.Casting, poursPath string) ([]domain.Material, error) {
	var materials []domain.Material
	for _, tmpl := range c.castings {
		m, err := c.forgeCasting(tmpl, &cfg, poursPath)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to forge")
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

func (c *kustomizeCasting) Cast(ctx context.Context, config installation.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Applying kustomize manifests")

	kustomizeDir := filepath.Join(poursPath, rootcasting.DeploymentDir)
	if _, err := os.Stat(filepath.Join(kustomizeDir, "kustomization.yaml")); os.IsNotExist(err) {
		return errors.Newf(errors.TypeNotFound, "kustomization.yaml does not exist at path: %s, run 'forge' first", kustomizeDir)
	}

	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := c.applyCRDs(runctx); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to apply CRDs")
	}

	cmd := exec.CommandContext(runctx, "kubectl", "apply", "-k", kustomizeDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	c.logger.DebugContext(runctx, "Running command",
		slog.String("command", fmt.Sprintf("kubectl apply -k %s", kustomizeDir)))

	if err := cmd.Run(); err != nil {
		c.logger.ErrorContext(runctx, "kubectl apply failed", slog.String("error", err.Error()))
		return errors.Wrapf(err, errors.TypeInternal, "kubectl apply -k failed")
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
			return errors.Wrapf(err, errors.TypeInternal, "failed to apply CRD %s", crd)
		}
	}

	return nil
}

func (c *kustomizeCasting) forgeCasting(tmpl *domain.Template, cfg *installation.Casting, poursPath string) ([]domain.Material, error) {
	templatePath := tmpl.Path()
	relPath := strings.TrimPrefix(templatePath, "templates/")
	relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))
	path := filepath.Join(rootcasting.DeploymentDir, relPath)
	material, err := tmpl.Render(cfg, path)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "render template %s", templatePath)
	}
	return []domain.Material{material}, nil
}

func getOverrideMaterials(config *installation.Casting) ([]domain.StructuredMaterial, error) {
	return renderStructured(config, []templateAt{
		{telemetryStoreOverrideTemplate, "store_overrides.yaml"},
	})
}

func getServiceMaterials(config *installation.Casting) ([]domain.StructuredMaterial, error) {
	return renderStructured(config, []templateAt{
		{clickhouseInstanceInstallation, "clickhouseInstallation.yaml"},
		{metastoreService, "metastoreServie.yaml"},
	})
}

type templateAt struct {
	tmpl *domain.Template
	path string
}

func renderStructured(config *installation.Casting, items []templateAt) ([]domain.StructuredMaterial, error) {
	materials := make([]domain.StructuredMaterial, 0, len(items))
	for _, item := range items {
		m, err := item.tmpl.Render(config, item.path)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "render template %s", item.tmpl.Path())
		}
		sm, ok := m.(domain.StructuredMaterial)
		if !ok {
			return nil, errors.Newf(errors.TypeInternal, "template %s does not produce a structured material", item.tmpl.Path())
		}
		materials = append(materials, sm)
	}
	return materials, nil
}
