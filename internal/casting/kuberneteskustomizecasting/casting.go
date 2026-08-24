package kuberneteskustomizecasting

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/kubectltooler"
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
			mcpDeployment,
			mcpService,
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
			mcpKustomization,
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

func (c *kustomizeCasting) Cast(ctx context.Context, config installation.Casting, poursPath string, toolers []tooler.Tooler) error {
	kubectl, err := kubectltooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := kubectltooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Dir:     filepath.Join(poursPath, rootcasting.DeploymentDir),
	}

	if enabled := config.Spec.TelemetryStore.Spec.Enabled; enabled != nil && *enabled {
		for _, crd := range clickhouseCRDs {
			release.URLs = append(release.URLs, fmt.Sprintf("https://raw.githubusercontent.com/Altinity/clickhouse-operator/%s/deploy/operatorhub/%s/%s", clickhouseOperatorVersion, clickhouseOperatorVersion, crd))
		}
	}

	return kubectl.Apply(ctx, release)
}

// Melt deletes what the kustomize root declares, the namespace included.
func (c *kustomizeCasting) Melt(ctx context.Context, config installation.Casting, poursPath string, toolers []tooler.Tooler) error {
	kubectl, err := kubectltooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := kubectltooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Dir:     filepath.Join(poursPath, rootcasting.DeploymentDir),
	}

	return kubectl.Delete(ctx, release)
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
