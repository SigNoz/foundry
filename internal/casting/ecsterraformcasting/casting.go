package ecsterraformcasting

import (
	"context"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/contract"
	awscontract "github.com/signoz/foundry/internal/contract/aws"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
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

	// One file per component, beside the molding config that component fetches
	// at task start. A component configured only through env has no configDir.
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

// Melt destroys everything this root's state records: the services, the task
// definitions and the config they read. What the substrate holds is another
// root's, and stays.
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

// templateData binds the casting to the substrate it runs on. Forge and the
// enricher each render from their own config; both come through here.
func (c *ecsCasting) templateData(config installation.Casting) (templateData, error) {
	name := config.Spec.Infrastructure.Name

	if name == "" {
		return templateData{}, errors.Newf(errors.TypeInvalidInput, "spec.infrastructure.name is not set: this casting finds its cluster, its subnets and its nodes by the substrate's own tags, so it has to be told which substrate it runs on")
	}

	substrate, err := contract.NewSubstrate(name)
	if err != nil {
		return templateData{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to resolve the substrate this installation runs on")
	}

	persistent := substrate.Select().WithStorage(contract.StorageClassPersistent)
	ephemeral := substrate.Select().WithStorage(contract.StorageClassEphemeral)

	// Every service uses awsvpc. A task needs a subnet, and never a public one.
	private := substrate.Select().WithSubnetType(contract.SubnetTypePrivate)

	return templateData{
		Casting: config,

		ClusterName:       awscontract.Cluster(substrate).Name(),
		SecurityGroupName: awscontract.SecurityGroup(substrate, awscontract.RoleTask).Name(),

		// Named after the workload. Several workloads share one substrate, and a
		// substrate-derived name collides on the second apply.
		TaskRoleName:      config.Metadata.Name + "-" + strings.ToLower(config.Kind().String()) + "-task",
		ExecutionRoleName: config.Metadata.Name + "-" + strings.ToLower(config.Kind().String()) + "-exec",

		VPCTags:    awscontract.Filter(substrate.Select()),
		SubnetTags: awscontract.Filter(private),
		NodeTags:   awscontract.Filter(persistent),

		ClaimTag:            awscontract.Tag(contract.TagKeyIdentities),
		PersistentPlacement: placement(persistent),
		EphemeralPlacement:  placement(ephemeral),
	}, nil
}

// templateData is what every template renders against. The casting is embedded,
// leaving `.Spec` and `.Metadata` as they were. The rest is derived from the
// substrate this installation is bound to.
type templateData struct {
	installation.Casting

	ClusterName       string
	SecurityGroupName string

	// TaskRoleName and ExecutionRoleName are this workload's own identity,
	// created and destroyed with this stack.
	TaskRoleName      string
	ExecutionRoleName string

	VPCTags    map[string]string
	SubnetTags map[string]string

	// NodeTags finds the substrate's persistent instances and the volumes
	// attached to them. The claim controller reads both.
	NodeTags map[string]string

	// ClaimTag records which identities hold a volume. It is written after
	// provisioning, and the Infrastructure casting does not reconcile it.
	ClaimTag string

	// Placement expressions in ECS' own syntax, matching the attributes a
	// container instance advertises.
	PersistentPlacement string
	EphemeralPlacement  string
}

// placement renders a selection as an ECS placement constraint. A container
// instance advertises exactly these attributes.
func placement(selection contract.Selection) string {
	filter := awscontract.Filter(selection)

	parts := make([]string, 0, len(filter))

	for _, key := range slices.Sorted(maps.Keys(filter)) {
		parts = append(parts, "attribute:"+key+" == "+filter[key])
	}

	return strings.Join(parts, " and ")
}

// getMaterials renders the component templates for the enricher to read service
// names back from. The template is the single source of node names.
func getMaterials(data templateData) ([]domain.StructuredMaterial, error) {
	var materials []domain.StructuredMaterial

	for _, tmpl := range []*domain.Template{
		mainTF,
		telemetryStoreTF,
		telemetryKeeperTF,
		metaStoreTF,
		signozTF,
		ingesterTF,
		mcpTF,
	} {
		m, err := tmpl.Render(data, tmpl.Path())
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to render material")
		}
		sm, ok := m.(domain.StructuredMaterial)
		if !ok {
			return nil, errors.Newf(errors.TypeInternal, "template %s does not produce a structured material", tmpl.Path())
		}
		materials = append(materials, sm)
	}

	return materials, nil
}
