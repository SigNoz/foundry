package ecsterraformcasting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
)

// The AppConfig agent writes the config into a task volume mounted here, and
// the collector reads it from the same place.
const configMount = "/conf"

type ecsCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *ecsCasting {
	return &ecsCasting{logger: logger}
}

func (c *ecsCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newEcsMoldingEnricher(), nil
}

func (c *ecsCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	data, err := c.templateData(config)
	if err != nil {
		return err
	}

	for _, tmpl := range []*domain.Template{versionsTF, providersTF, backendTF, variablesTF, tfvarsTF, mainTF, collectorTF} {
		material, err := tmpl.Render(data, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
		if err != nil {
			return err
		}

		p.AddJSON(material.FmtContents(), material.Path())
	}

	// AppConfig reads the config off disk at plan time, so the pour is the
	// source the hosted configuration version is built from.
	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *ecsCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	terraform, err := terraformtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	return terraform.Apply(ctx, release(config, outputPath, p))
}

// Melt destroys what this root's state records: the daemon service, its task
// definition and the AppConfig application holding its config.
func (c *ecsCasting) Melt(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	terraform, err := terraformtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	return terraform.Destroy(ctx, release(config, outputPath, p))
}

func release(config collectionagent.Casting, outputPath string, p *pourer.Pourer) terraformtooler.Release {
	return terraformtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Root:    filepath.Join(outputPath, p.Dir()),
	}
}

// templateData resolves every identifier the templates render, so a template
// never reaches into the annotations itself.
func (c *ecsCasting) templateData(config collectionagent.Casting) (templateData, error) {
	annotations := config.Metadata.Annotations

	region := collectionagent.ECSRegion.Resolve(annotations)
	if region == "" {
		return templateData{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "no region is stated: state the %q annotation", collectionagent.ECSRegion.Key)
	}

	cluster := Reference{Stated: collectionagent.ECSClusterARN.Resolve(annotations)}
	if !cluster.IsStated() {
		return templateData{}, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "no cluster is stated: state the %q annotation", collectionagent.ECSClusterARN.Key)
	}

	// The roles die with this stack, so the workload names them itself. Several
	// workloads share one cluster, and a cluster-derived name collides on the
	// second apply.
	workload := config.Metadata.Name + "-" + strings.ToLower(config.Kind().String())

	configKey := config.Spec.Collector.Kind.ConfigKey()

	return templateData{
		Casting: config,
		Region:  region,
		Cluster: cluster,
		TaskRole: Reference{
			Stated: collectionagent.ECSTaskRoleARN.Resolve(annotations),
			Name:   workload + "-task",
		},
		ExecutionRole: Reference{
			Stated: collectionagent.ECSTaskExecutionRoleARN.Resolve(annotations),
			Name:   workload + "-exec",
		},
		Application: workload,
		Environment: "default",
		Profile:     strings.ReplaceAll(filepath.Dir(configKey), "/", "-"),
		Source:      configKey,
		Target:      filepath.Join(configMount, filepath.Base(configKey)),
		Digest:      digest(config.Spec.Collector.Spec.Config.Data[configKey]),
		ConfigMount: configMount,
	}, nil
}

func digest(content string) string {
	sum := sha256.Sum256([]byte(content))

	return hex.EncodeToString(sum[:])
}

// Reference is one identifier, resolved. An operator states the object and it
// is referenced as-is; otherwise the two roles are created under the workload's
// own name. Exactly one side is populated.
type Reference struct {
	Stated string
	Name   string
}

func (r Reference) IsStated() bool {
	return r.Stated != ""
}

// templateData embeds the casting, so `.Spec` and `.Metadata` stay as they
// were; the rest is what the templates would otherwise have to derive.
type templateData struct {
	collectionagent.Casting

	Region string

	Cluster Reference

	// The roles are the workload's own identity, created and destroyed with
	// this stack.
	TaskRole      Reference
	ExecutionRole Reference

	// Application carries the Kind, so a CollectionAgent and an Installation of
	// the same metadata.name do not collide on one account.
	Application string
	Environment string

	// Profile is the AppConfig configuration profile holding the collector
	// config. Source is where the pour keeps it, relative to the root; Target
	// is where the AppConfig agent writes it inside the task.
	Profile string
	Source  string
	Target  string

	// Digest changes with the config, and a task definition that changes
	// replaces the task. The collector reads its config once at start, so an
	// in-place rewrite would not reach it.
	Digest string

	ConfigMount string
}
