package ecsterraformcasting

import (
	"context"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/contract"
	awscontract "github.com/signoz/foundry/internal/contract/aws"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
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
		{config.Spec.MetaStore.Spec.IsEnabled(), "metastore.tf.json", metaStoreTF, filepath.Join("metastore", config.Spec.MetaStore.Kind.String()), config.Spec.MetaStore.Spec.Config.Data},
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

func (c *ecsCasting) Cast(ctx context.Context, config installation.Casting, outputPath string, toolers []tooler.Tooler) error {
	terraform, err := terraformtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	return terraform.Apply(ctx, c.release(config, outputPath))
}

// Melt destroys what this root's state records: the services, the task
// definitions and their config. What the substrate holds is another root's.
func (c *ecsCasting) Melt(ctx context.Context, config installation.Casting, outputPath string, toolers []tooler.Tooler) error {
	terraform, err := terraformtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	return terraform.Destroy(ctx, c.release(config, outputPath))
}

func (c *ecsCasting) release(config installation.Casting, outputPath string) terraformtooler.Release {
	return terraformtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Root:    filepath.Join(outputPath, rootcasting.DeploymentDir),
	}
}

// Forge and the enricher each render from their own config, both through here.
func (c *ecsCasting) templateData(config installation.Casting) (templateData, error) {
	annotations := config.Metadata.Annotations

	region := installation.ECSRegion.Resolve(annotations)
	if region == "" {
		return templateData{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "no region is stated: state the %q annotation", installation.ECSRegion.Key)
	}

	data := templateData{
		Casting:   config,
		Substrate: config.Spec.Infrastructure.Name,
		Region:    region,

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

	// Without a substrate the four axes below have no derived side, so each one
	// has to be stated, and no seat or placement is rendered at all.
	if data.Substrate == "" {
		for annotation, reference := range map[v1alpha1.Annotation]Reference{
			installation.ECSClusterARN:       data.Cluster,
			installation.ECSVPCID:            data.VPC,
			installation.ECSSubnetIDs:        data.Subnets,
			installation.ECSSecurityGroupIDs: data.SecurityGroup,
		} {
			if !reference.IsStated() {
				return templateData{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "no infrastructure is stated, so %q is required: without a substrate there is nothing to find it by", annotation.Key)
			}
		}

		return data, nil
	}

	substrate, err := contract.NewSubstrate(data.Substrate)
	if err != nil {
		return templateData{}, foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to resolve the substrate this installation runs on")
	}

	persistent := substrate.Select().WithStorage(contract.StorageClassPersistent)
	ephemeral := substrate.Select().WithStorage(contract.StorageClassEphemeral)

	// Every service uses awsvpc. A task needs a subnet, and never a public one.
	private := substrate.Select().WithSubnetType(contract.SubnetTypePrivate)

	data.Cluster.Name = awscontract.Cluster(substrate).Name()
	data.SecurityGroup.Name = awscontract.SecurityGroup(substrate, awscontract.RoleTask).Name()
	data.VPC.Tags = awscontract.Filter(substrate.Select())
	data.Subnets.Tags = awscontract.Filter(private)

	data.NodeTags = awscontract.Filter(persistent)
	data.ClaimTag = awscontract.Tag(contract.TagKeyIdentities)
	data.PersistentPlacement = placement(persistent)
	data.EphemeralPlacement = placement(ephemeral)

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

// Reference is one rendezvous axis, resolved. An operator states the object and
// it is referenced as-is; otherwise it is found by the substrate's tags, and
// the templates name what this stack creates. Axes that name a list state
// through StatedIDs, the rest through Stated.
type Reference struct {
	Stated    string
	StatedIDs []string
	Name      string
	Tags      map[string]string
}

func (r Reference) IsStated() bool {
	return r.Stated != "" || len(r.StatedIDs) > 0
}

// templateData embeds the casting, so `.Spec` and `.Metadata` stay as they
// were; the rest resolves each axis to a stated or a derived side.
type templateData struct {
	installation.Casting

	// Substrate is the name this installation is bound to, empty when the
	// operator brought their own cluster. What only a substrate can derive is
	// empty with it.
	Substrate string

	Region string

	Cluster       Reference
	VPC           Reference
	Subnets       Reference
	SecurityGroup Reference

	// The roles are the workload's own identity, created and destroyed with
	// this stack, so they derive from the casting and never the substrate.
	TaskRole      Reference
	ExecutionRole Reference

	// NodeTags finds the substrate's persistent instances and the volumes
	// attached to them. The claim controller reads both.
	NodeTags map[string]string

	// ClaimTag records which identities hold a volume. It is written after
	// provisioning, and the Infrastructure casting does not reconcile it.
	ClaimTag string

	PersistentPlacement string
	EphemeralPlacement  string
}

// placement renders a selection as the attributes a container instance advertises.
func placement(selection contract.Selection) string {
	filter := awscontract.Filter(selection)

	parts := make([]string, 0, len(filter))

	for _, key := range slices.Sorted(maps.Keys(filter)) {
		parts = append(parts, "attribute:"+key+" == "+filter[key])
	}

	return strings.Join(parts, " and ")
}

// getMaterials renders the component templates; they are the source of node
// names. Keyed by molding kind, so the enricher reaches a component by the
// same value it switches on.
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
