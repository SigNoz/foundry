package ecsterraformcasting

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ rootcasting.Casting = (*ecsCasting)(nil)

type ecsCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *ecsCasting {
	return &ecsCasting{
		logger: logger,
	}
}

func (c *ecsCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	data, err := c.templateData(*config)
	if err != nil {
		return nil, err
	}

	return newEcsMoldingEnricher(data)
}

func (c *ecsCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {
	var materials []domain.Material

	dir := rootcasting.DeploymentDir

	data, err := c.templateData(config)
	if err != nil {
		return nil, err
	}

	for filename, tmpl := range map[string]*domain.Template{
		"versions.tf.json":      versionsTF,
		"backend.tf.json":       backendTF,
		"providers.tf.json":     providersTF,
		"main.tf.json":          mainTF,
		"variables.tf.json":     variablesTF,
		"outputs.tf.json":       outputsTF,
		"terraform.tfvars.json": tfarsTF,
	} {
		m, err := tmpl.Render(data, filepath.Join(dir, filename))
		if err != nil {
			return nil, err
		}

		materials = append(materials, m)
	}

	// A component configured only through env has no configDir.
	components := []struct {
		enabled   bool
		filename  string
		template  *domain.Template
		configDir string
		config    map[string]string
	}{
		{config.Spec.TelemetryKeeper.Spec.IsEnabled(), "telemetrykeeper.tf.json", telemetryKeeperTF, filepath.Join("telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String()), config.Spec.TelemetryKeeper.Spec.Config.Data},
		{config.Spec.TelemetryStore.Spec.IsEnabled(), "telemetrystore.tf.json", telemetryStoreTF, filepath.Join("telemetrystore", config.Spec.TelemetryStore.Kind.String()), config.Spec.TelemetryStore.Spec.Config.Data},
		{config.Spec.TelemetryStore.Spec.IsEnabled(), "telemetrystore_migrator.tf.json", migratorTF, "", nil},
		{config.Spec.MetaStore.Spec.IsEnabled() && config.Spec.MetaStore.Kind == installation.MetaStoreKindPostgres, "metastore.tf.json", metaStoreTF, filepath.Join("metastore", config.Spec.MetaStore.Kind.String()), config.Spec.MetaStore.Spec.Config.Data},
		{config.Spec.Signoz.Spec.IsEnabled(), "signoz.tf.json", signozTF, "", nil},
		{config.Spec.Ingester.Spec.IsEnabled(), "ingester.tf.json", ingesterTF, "ingester", config.Spec.Ingester.Spec.Config.Data},
		{config.Spec.MCP.Spec.IsEnabled(), "mcp.tf.json", mcpTF, "", nil},
	}

	for _, component := range components {
		if !component.enabled {
			continue
		}

		m, err := component.template.Render(data, filepath.Join(dir, component.filename))
		if err != nil {
			return nil, err
		}

		materials = append(materials, m)

		for filename, content := range component.config {
			material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(dir, component.configDir, filename))
			if err != nil {
				return nil, err
			}

			materials = append(materials, material)
		}
	}

	return materials, nil
}

const planFile = "tfplan"

func (c *ecsCasting) Cast(ctx context.Context, config installation.Casting, outputPath string) error {
	root := filepath.Join(outputPath, rootcasting.DeploymentDir)

	if err := c.terraform(ctx, root, "init"); err != nil {
		return err
	}

	if err := c.terraform(ctx, root, "plan", "-out="+planFile); err != nil {
		return err
	}

	return c.terraform(ctx, root, "apply", planFile)
}

func (c *ecsCasting) terraform(ctx context.Context, root, verb string, args ...string) error {
	argv := append([]string{"-chdir=" + root, verb}, args...)

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "terraform "+strings.Join(argv, " ")))

	// Stdout is foundry's own contract, so the tool's output goes to stderr.
	cmd := exec.CommandContext(ctx, "terraform", argv...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run terraform %s", verb)
	}

	return nil
}

func (c *ecsCasting) templateData(config installation.Casting) (templateData, error) {
	annotations := config.Metadata.Annotations

	data := templateData{
		Casting: config,
		Region:  installation.ECSRegion.Resolve(annotations),

		Cluster:       Reference{Stated: installation.ECSClusterARN.Resolve(annotations)},
		VPC:           Reference{Stated: installation.ECSVPCID.Resolve(annotations)},
		TaskRole:      Reference{Stated: installation.ECSTaskRoleARN.Resolve(annotations)},
		ExecutionRole: Reference{Stated: installation.ECSTaskExecutionRoleARN.Resolve(annotations)},
	}

	subnets, err := statedIDs(installation.ECSSubnetIDs, annotations)
	if err != nil {
		return templateData{}, err
	}

	data.Subnets = Reference{StatedIDs: subnets}

	securityGroups, err := statedIDs(installation.ECSSecurityGroupIDs, annotations)
	if err != nil {
		return templateData{}, err
	}

	data.SecurityGroup = Reference{StatedIDs: securityGroups}

	return data, nil
}

// An annotation that is set but names nothing is a mistake, not an omission.
func statedIDs(annotation v1alpha1.Annotation, annotations map[string]string) ([]string, error) {
	value := annotation.Resolve(annotations)
	if value == "" {
		return nil, nil
	}

	ids := make([]string, 0, 1)
	for id := range strings.SplitSeq(value, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to parse the %q annotation: no ids found in %q", annotation.Key, value)
	}

	return ids, nil
}

// Reference is one object the casting names: lists through StatedIDs, the
// rest through Stated.
type Reference struct {
	Stated    string
	StatedIDs []string
}

func (r Reference) IsStated() bool {
	return r.Stated != "" || len(r.StatedIDs) > 0
}

// templateData embeds the casting, so `.Spec` and `.Metadata` stay as they were.
type templateData struct {
	installation.Casting

	Region string

	Cluster       Reference
	VPC           Reference
	Subnets       Reference
	SecurityGroup Reference

	// The roles are this stack's own identity, so an absent one is created.
	TaskRole      Reference
	ExecutionRole Reference
}

// getMaterials renders the component templates, keyed by the molding kind the
// enricher switches on.
func getMaterials(data templateData) (map[v1alpha1.MoldingKind]domain.StructuredMaterial, error) {
	materials := map[v1alpha1.MoldingKind]domain.StructuredMaterial{}

	for kind, tmpl := range map[v1alpha1.MoldingKind]*domain.Template{
		v1alpha1.MoldingKindTelemetryStore:  telemetryStoreTF,
		v1alpha1.MoldingKindTelemetryKeeper: telemetryKeeperTF,
		v1alpha1.MoldingKindMetaStore:       metaStoreTF,
		v1alpha1.MoldingKindSignoz:          signozTF,
		v1alpha1.MoldingKindIngester:        ingesterTF,
		v1alpha1.MoldingKindMCP:             mcpTF,
	} {
		m, err := tmpl.Render(data, tmpl.Path())
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to render material")
		}

		sm, ok := m.(domain.StructuredMaterial)
		if !ok {
			return nil, foundryerrors.Newf(foundryerrors.TypeInternal, "template %q does not produce a structured material", tmpl.Path())
		}

		materials[kind] = sm
	}

	return materials, nil
}
