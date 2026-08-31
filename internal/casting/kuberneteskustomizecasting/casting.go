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
	"github.com/signoz/foundry/internal/tooler/kubetooler"
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
		clickhouseOperatorCHICRD,
		clickhouseOperatorCHITCRD,
		clickhouseOperatorConfigCRD,
		clickhouseOperatorCHKCRD,
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

// Cast applies the operators tier before the root: one pass would post the
// ClickHouseInstallation before its kind exists.
func (c *kustomizeCasting) Cast(ctx context.Context, config installation.Casting, poursPath string, toolers []tooler.Tooler) error {
	kube, err := kubetooler.Lookup(toolers)
	if err != nil {
		return err
	}

	c.logger.InfoContext(ctx, "applying kustomize manifests",
		slog.String("release", config.Metadata.Name),
		slog.String("namespace", config.Metadata.Name),
	)

	if needsClickhouseOperator(&config) {
		operators := c.release(config, poursPath)
		operators.Dir = filepath.Join(operators.Dir, "operators", "clickhouse-operator")

		if err := kube.Apply(ctx, operators); err != nil {
			return err
		}
	}

	// A Job's pod template is immutable: the finished migrator run is replaced,
	// not patched, or a re-cast with a changed image would be rejected.
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		migrator := c.release(config, poursPath)
		migrator.Dir = filepath.Join(migrator.Dir, "telemetrystore-migrator")

		if err := kube.Delete(ctx, migrator); err != nil {
			return err
		}
	}

	return kube.Apply(ctx, c.release(config, poursPath))
}

// Melt leaves the namespace and the definitions standing: the tooler keeps
// what carries data or is shared with every other release in the cluster.
func (c *kustomizeCasting) Melt(ctx context.Context, config installation.Casting, poursPath string, toolers []tooler.Tooler) error {
	kube, err := kubetooler.Lookup(toolers)
	if err != nil {
		return err
	}

	return kube.Delete(ctx, c.release(config, poursPath))
}

func (c *kustomizeCasting) release(config installation.Casting, poursPath string) kubetooler.Release {
	return kubetooler.Release{
		Release:   domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Namespace: config.Metadata.Name,
		Dir:       filepath.Join(poursPath, rootcasting.DeploymentDir),

		// The Kind, not the release, so an Installation's apply never contends
		// with a CollectionAgent's over an object they share.
		FieldManager: "foundry-" + strings.ToLower(config.Kind().String()),
	}
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
