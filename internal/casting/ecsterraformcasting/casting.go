package ecsterraformcasting

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

	// A component configured only through env has no configDir. The metastore
	// is a service only under postgres: sqlite is a file the signoz task holds.
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

func (c *ecsCasting) Cast(ctx context.Context, config installation.Casting, outputPath string) error {
	c.logger.InfoContext(ctx, "Running Terraform for ECS deployment")

	deploymentDir := filepath.Join(outputPath, rootcasting.DeploymentDir)

	// Verify terraform files exist
	if _, err := os.Stat(filepath.Join(deploymentDir, "main.tf.json")); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "terraform files do not exist at path: %s; run forge first", deploymentDir)
	}

	// Create a context with 10-minute timeout (terraform can be slow)
	runctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Run terraform init
	c.logger.InfoContext(runctx, "Running terraform init")
	initCmd := exec.CommandContext(runctx, "terraform", "-chdir="+deploymentDir, "init")
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		c.logger.ErrorContext(runctx, "terraform init failed", slog.String("error", err.Error()))
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "terraform init failed")
	}

	// Run terraform apply
	c.logger.InfoContext(runctx, "Running terraform apply")
	args := []string{"-chdir=" + deploymentDir, "apply", "-auto-approve"}
	c.logger.DebugContext(runctx, "Running command", slog.String("command", "terraform "+strings.Join(args, " ")))

	applyCmd := exec.CommandContext(runctx, "terraform", args...)
	applyCmd.Stdout = os.Stdout
	applyCmd.Stderr = os.Stderr
	if err := applyCmd.Run(); err != nil {
		c.logger.ErrorContext(runctx, "terraform apply failed", slog.String("error", err.Error()))
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "terraform apply failed")
	}

	c.logger.InfoContext(runctx, "Terraform apply completed successfully")
	return nil
}

// Forge and the enricher each render from their own config, both through here.
func (c *ecsCasting) templateData(config installation.Casting) (templateData, error) {
	annotations := config.Metadata.Annotations

	data := templateData{
		Casting: config,

		Region:        installation.ECSRegion.Resolve(annotations),
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

	if data.Region == "" {
		return templateData{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "no region is stated: state the %q annotation", installation.ECSRegion.Key)
	}

	// Nothing here finds an object by the substrate's tags yet, so a binding
	// would be read by nothing. Refuse it rather than ignore it.
	if config.Spec.Infrastructure.Name != "" {
		return templateData{}, foundryerrors.Newf(foundryerrors.TypeUnsupported, "failed to bind the substrate %q: this casting places tasks onto a cluster it does not provision, so state every object the cluster is made of instead", config.Spec.Infrastructure.Name)
	}

	// The cluster is placed onto, never provisioned, so every object it is made
	// of has to be named. The two roles are the workload's own identity and are
	// created with this stack when unstated.
	for annotation, reference := range map[v1alpha1.Annotation]Reference{
		installation.ECSClusterARN:       data.Cluster,
		installation.ECSVPCID:            data.VPC,
		installation.ECSSubnetIDs:        data.Subnets,
		installation.ECSSecurityGroupIDs: data.SecurityGroup,
	} {
		if !reference.IsStated() {
			return templateData{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "no cluster object is stated for %q: state it, and every other object the cluster is made of", annotation.Key)
		}
	}

	return data, nil
}

// statedIDs reads a comma-separated annotation. An annotation that is set but
// names nothing is a mistake, not an omission.
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

// Reference is one rendezvous axis, resolved. Axes that name a list state
// through StatedIDs, the rest through Stated.
type Reference struct {
	Stated    string
	StatedIDs []string
}

func (r Reference) IsStated() bool {
	return r.Stated != "" || len(r.StatedIDs) > 0
}

// templateData embeds the casting, so `.Spec` and `.Metadata` stay as they
// were; the rest is what the templates would otherwise resolve themselves.
type templateData struct {
	installation.Casting

	Region string

	Cluster       Reference
	VPC           Reference
	Subnets       Reference
	SecurityGroup Reference

	// The roles are the workload's own identity, created and destroyed with
	// this stack, so an absent one is created rather than looked up.
	TaskRole      Reference
	ExecutionRole Reference
}

// getMaterials renders the component templates; they are the source of service
// names. Keyed by molding kind, so the enricher reaches a component by the same
// value it switches on.
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
