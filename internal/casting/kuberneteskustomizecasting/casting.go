package kuberneteskustomizecasting

import (
	"context"
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
	logger *slog.Logger
}

func New(logger *slog.Logger) *kustomizeCasting {
	return &kustomizeCasting{logger: logger}
}

var (
	deploymentTemplates = []*domain.Template{
		deploymentNamespace,
		deploymentKustomization,
	}
	clickhouseOperatorTemplates = []*domain.Template{
		clickhouseOperatorClusterrole,
		clickhouseOperatorClusterrolebinding,
		clickhouseOperatorConfigmap,
		clickhouseOperatorDeployment,
		clickhouseOperatorService,
		clickhouseOperatorServiceaccount,
		clickhouseOperatorKustomization,
		clickhouseOperatorNamespace,
	}
	telemetryStoreTemplates = []*domain.Template{
		clickhouseInstanceInstallation,
		clickhouseInstanceConfigmap,
		clickhouseInstallationKustomization,
		telemetrystoreMigratorJob,
		telemetrystoreMigratorKustomization,
	}
	telemetryKeeperTemplates = []*domain.Template{
		clickhouseKeeperInstallation,
		clickhouseKeeperKustomization,
	}
	metaStoreTemplates = []*domain.Template{
		metastoreService,
		metastoreServiceaccount,
		metastoreStatefulset,
		metastoreKustomization,
	}
	signozTemplates = []*domain.Template{
		signozService,
		signozServiceaccount,
		signozStatefulset,
		signozKustomization,
	}
	mcpTemplates = []*domain.Template{
		mcpDeployment,
		mcpService,
		mcpKustomization,
	}
	ingesterTemplates = []*domain.Template{
		ingesterConfigmap,
		ingesterDeployment,
		ingesterService,
		ingesterServiceaccount,
		ingesterKustomization,
	}
)

func (c *kustomizeCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newKustomizeMoldingEnricher(config)
}

func (c *kustomizeCasting) Forge(ctx context.Context, cfg installation.Casting, poursPath string) ([]domain.Material, error) {
	templates := append([]*domain.Template{}, deploymentTemplates...)

	if needsClickhouseOperator(&cfg) {
		templates = append(templates, clickhouseOperatorTemplates...)
	}

	if cfg.Spec.TelemetryStore.Spec.IsEnabled() {
		templates = append(templates, telemetryStoreTemplates...)
	}

	if cfg.Spec.TelemetryKeeper.Spec.IsEnabled() {
		templates = append(templates, telemetryKeeperTemplates...)
	}

	if cfg.Spec.MetaStore.Spec.IsEnabled() && cfg.Spec.MetaStore.Kind == installation.MetaStoreKindPostgres {
		templates = append(templates, metaStoreTemplates...)
	}

	if cfg.Spec.Signoz.Spec.IsEnabled() {
		templates = append(templates, signozTemplates...)
	}

	if cfg.Spec.MCP.Spec.IsEnabled() {
		templates = append(templates, mcpTemplates...)
	}

	if cfg.Spec.Ingester.Spec.IsEnabled() {
		templates = append(templates, ingesterTemplates...)
	}

	var materials []domain.Material
	for _, tmpl := range templates {
		m, err := c.forgeCasting(tmpl, &cfg)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to forge")
		}
		materials = append(materials, m...)
	}
	return materials, nil
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

	// The operators tier goes first: one pass would post the
	// ClickHouseInstallation before its kind exists.
	if needsClickhouseOperator(&config) {
		operators := release
		operators.Dir = filepath.Join(release.Dir, "operators", "clickhouse-operator")

		if err := kubectl.Apply(ctx, operators); err != nil {
			return err
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

// The Altinity operator serves both the CHI and the CHK.
func needsClickhouseOperator(cfg *installation.Casting) bool {
	return cfg.Spec.TelemetryStore.Spec.IsEnabled() ||
		(cfg.Spec.TelemetryKeeper.Spec.IsEnabled() && cfg.Spec.TelemetryKeeper.Kind == installation.TelemetryKeeperKindClickhouseKeeper)
}

func (c *kustomizeCasting) forgeCasting(tmpl *domain.Template, cfg *installation.Casting) ([]domain.Material, error) {
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
