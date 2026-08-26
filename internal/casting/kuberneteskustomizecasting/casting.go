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

// operators/ is its own tier, outside the root kustomization: it is applied
// first and its CRDs waited on, since one pass would post the
// ClickHouseInstallation before its kind exists.
var clickhouseCRDs = []string{
	"clickhouseinstallations.clickhouse.altinity.com",
	"clickhouseinstallationtemplates.clickhouse.altinity.com",
	"clickhouseoperatorconfigurations.clickhouse.altinity.com",
	"clickhousekeeperinstallations.clickhouse-keeper.altinity.com",
}

func (c *kustomizeCasting) Cast(ctx context.Context, config installation.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Applying kustomize manifests")

	kustomizeDir := filepath.Join(poursPath, rootcasting.DeploymentDir)
	if _, err := os.Stat(filepath.Join(kustomizeDir, "kustomization.yaml")); os.IsNotExist(err) {
		return errors.Newf(errors.TypeNotFound, "kustomization.yaml does not exist at path: %s, run 'forge' first", kustomizeDir)
	}

	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if needsClickhouseOperator(&config) {
		if err := c.kubectl(runctx, "apply", "-k", filepath.Join(kustomizeDir, "operators", "clickhouse-operator")); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to apply clickhouse-operator")
		}

		args := []string{"wait", "--for=condition=Established", "--timeout=60s"}
		for _, crd := range clickhouseCRDs {
			args = append(args, "crd/"+crd)
		}
		if err := c.kubectl(runctx, args...); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed waiting for clickhouse CRDs to be established")
		}
	}

	// A Job's pod template is immutable, so a re-cast with a changed migrator
	// (image, DSN) would be rejected; the finished run is replaced, not patched.
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		job := config.Metadata.Name + "-telemetrystore-migrator"
		if err := c.kubectl(runctx, "delete", "job", job, "--namespace", config.Metadata.Name, "--ignore-not-found"); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to delete job %q", job)
		}
	}

	if err := c.kubectl(runctx, "apply", "-k", kustomizeDir); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "kubectl apply -k failed")
	}

	c.logger.InfoContext(runctx, "Kustomize manifests applied successfully")
	return nil
}

func (c *kustomizeCasting) kubectl(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	c.logger.DebugContext(ctx, "Running command",
		slog.String("command", fmt.Sprintf("kubectl %s", strings.Join(args, " "))))

	if err := cmd.Run(); err != nil {
		c.logger.ErrorContext(ctx, "kubectl failed", slog.String("error", err.Error()))
		return err
	}
	return nil
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
