package ecsterraformcasting

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/contract"
	"github.com/signoz/foundry/internal/contract/aws"
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

	objects := []struct {
		key        string
		annotation v1alpha1.Annotation
		list       bool
		cluster    bool
	}{
		{"ClusterARN", installation.ECSClusterARN, false, true},
		{"VPCID", installation.ECSVPCID, false, true},
		{"SubnetIDs", installation.ECSPrivateSubnetIDs, true, true},
		{"SecurityGroupIDs", installation.ECSSecurityGroupIDs, true, true},
		{"TaskRoleARN", installation.ECSTaskRoleARN, false, false},
		{"ExecutionRoleARN", installation.ECSTaskExecutionRoleARN, false, false},
	}

	data := templateData{
		Casting: config,
		Region:  installation.ECSRegion.Resolve(annotations),
		Stated:  map[string]any{},
	}

	for _, object := range objects {
		if object.list {
			ids, err := statedIDs(object.annotation, annotations)
			if err != nil {
				return templateData{}, err
			}

			if len(ids) > 0 {
				data.Stated[object.key] = ids
			}

			continue
		}

		if value := object.annotation.Resolve(annotations); value != "" {
			data.Stated[object.key] = value
		}
	}

	name := config.Spec.Infrastructure.Name

	if name == "" {
		return data, nil
	}

	for _, object := range objects {
		if _, ok := data.Stated[object.key]; ok && object.cluster {
			return templateData{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "the installation is bound to infrastructure %q and states the %q annotation: state none of the cluster annotations, or unbind and state all four", name, object.annotation.Key)
		}
	}

	bound, err := contract.NewSubstrate(name)
	if err != nil {
		return templateData{}, foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to resolve the infrastructure the installation is bound to")
	}

	persistent, ephemeral := contract.StorageClassPersistent.String(), contract.StorageClassEphemeral.String()

	// Sqlite is a file the signoz task holds, so it needs a persistent node.
	signoz := ephemeral
	if config.Spec.MetaStore.Kind == installation.MetaStoreKindSQLite {
		signoz = persistent
	}

	// Spelled and counted as each component template names its nodes, so every service finds its own seat.
	nodes := []struct {
		enabled  bool
		prefix   string
		replicas *int
	}{
		{config.Spec.TelemetryKeeper.Spec.IsEnabled(), "telemetrykeeper-" + config.Spec.TelemetryKeeper.Kind.String(), config.Spec.TelemetryKeeper.Spec.Cluster.Replicas},
		{config.Spec.MetaStore.Spec.IsEnabled() && config.Spec.MetaStore.Kind == installation.MetaStoreKindPostgres, "metastore-" + config.Spec.MetaStore.Kind.String(), config.Spec.MetaStore.Spec.Cluster.Replicas},
		{config.Spec.Signoz.Spec.IsEnabled() && signoz == persistent, "signoz", config.Spec.Signoz.Spec.Cluster.Replicas},
	}

	identities := []string{}

	for _, node := range nodes {
		if !node.enabled {
			continue
		}

		count := 1
		if node.replicas != nil {
			count = max(1, *node.replicas)
		}

		for i := range count {
			identities = append(identities, fmt.Sprintf("%s-%d", node.prefix, i))
		}
	}

	if store := config.Spec.TelemetryStore; store.Spec.IsEnabled() {
		shards := 1
		if store.Spec.Cluster.Shards != nil {
			shards = max(1, *store.Spec.Cluster.Shards)
		}

		perShard := 1
		if store.Spec.Cluster.Replicas != nil {
			perShard = *store.Spec.Cluster.Replicas + 1
		}

		for s := range shards {
			for r := range perShard {
				identities = append(identities, fmt.Sprintf("telemetrystore-%s-%d-%d", store.Kind, s, r))
			}
		}
	}

	data.Substrate = map[string]any{
		"ClusterName":       aws.Cluster(bound).Name(),
		"VPC":               aws.VPC(bound).Filter(),
		"Subnets":           aws.Filter(bound.Select().WithSubnetType(contract.SubnetTypePrivate)),
		"SecurityGroupName": aws.SecurityGroup(bound, aws.RoleTask).Name(),
		"SecurityGroup":     aws.SecurityGroup(bound, aws.RoleTask).Filter(),
		"Persistent":        aws.Filter(bound.Select().WithStorage(contract.StorageClassPersistent)),
		"StorageKey":        aws.Tag(contract.TagKeyStorage),
		"IdentitiesKey":     aws.Tag(contract.TagKeyIdentities),
		"Identities":        identities,
		"Storage": map[string]string{
			v1alpha1.MoldingKindTelemetryStore.String():  persistent,
			v1alpha1.MoldingKindTelemetryKeeper.String(): persistent,
			v1alpha1.MoldingKindMetaStore.String():       persistent,
			v1alpha1.MoldingKindSignoz.String():          signoz,
			v1alpha1.MoldingKindIngester.String():        ephemeral,
			v1alpha1.MoldingKindMCP.String():             ephemeral,
		},
	}

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

// templateData embeds the casting, so `.Spec` and `.Metadata` stay as they were.
type templateData struct {
	installation.Casting

	Region string

	// What the operator states: the cluster objects (none when bound) and the roles (created when absent).
	Stated map[string]any

	// Nil unless spec.infrastructure names the substrate.
	Substrate map[string]any
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
