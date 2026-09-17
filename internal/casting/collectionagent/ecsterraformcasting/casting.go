package ecsterraformcasting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

// The sidecar writes the config here and the collector reads it.
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
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindAgent:
		if err := c.forgeAgent(config, p); err != nil {
			return err
		}
	case collectionagent.CollectorKindSidecar:
		if err := c.forgeSidecar(config, p); err != nil {
			return err
		}
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	// Terraform reads the config off disk at plan time, so the pour is the
	// source the delivered configuration is built from.
	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *ecsCasting) forgeAgent(config collectionagent.Casting, p *pourer.Pourer) error {
	data := c.templateData(config)

	for _, tmpl := range []*domain.Template{versionsTF, providersTF, backendTF, variablesTF, tfvarsTF, mainTF, collectorTF} {
		material, err := tmpl.Render(data, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
		if err != nil {
			return err
		}

		p.AddJSON(material.FmtContents(), material.Path())
	}

	return nil
}

func (c *ecsCasting) forgeSidecar(config collectionagent.Casting, p *pourer.Pourer) error {
	replicas := 1
	if cluster := config.Spec.Collector.Spec.Cluster; cluster.Replicas != nil {
		replicas = *cluster.Replicas
	}

	if replicas != 1 {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to forge the sidecar module: spec.collector.spec.cluster.replicas is %d, a sidecar runs once in every task of the application it joins", replicas)
	}

	data := sidecarTemplateDataFor(config)

	for _, tmpl := range []*domain.Template{sidecarVersionsTF, sidecarVariablesTF, sidecarMainTF, sidecarOutputsTF} {
		material, err := tmpl.Render(data, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
		if err != nil {
			return err
		}

		p.AddJSON(material.FmtContents(), material.Path())
	}

	return nil
}

const planFile = "tfplan"

func (c *ecsCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	root := filepath.Join(outputPath, p.Dir())

	if config.Spec.Collector.Kind == collectionagent.CollectorKindSidecar {
		c.castSidecar(ctx, root)

		return nil
	}

	if err := c.terraform(ctx, root, "init"); err != nil {
		return err
	}

	if err := c.terraform(ctx, root, "plan", "-out="+planFile); err != nil {
		return err
	}

	return c.terraform(ctx, root, "apply", planFile)
}

// A sidecar lives in a task definition foundry neither owns nor rewrites, so
// the pour is a module and the apply is the operator's own.
func (c *ecsCasting) castSidecar(ctx context.Context, root string) {
	// Terraform reads a local module source as a path relative to the root
	// that imports it, and only with the ./ prefix.
	source := root
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, root); err == nil {
			source = "./" + rel
		}
	}

	c.logger.InfoContext(ctx, "The sidecar collector is a terraform module your own terraform imports, so foundry applies nothing",
		slog.String("module", root))
	c.logger.InfoContext(ctx, "Import the module and name the execution role of the task definition the collector joins",
		slog.String("block", `module "signoz_sidecar" { source = "`+source+`" }`),
		slog.String("input", "execution_role_name = <the task definition's execution role name>"))
	c.logger.InfoContext(ctx, "Add the collector to the task definition",
		slog.String("container_definitions", "jsonencode(concat([module.signoz_sidecar.container_definition], local.containers))"))
	c.logger.InfoContext(ctx, "Ship the application containers' logs to the collector",
		slog.String("log_configuration", "module.signoz_sidecar.log_configuration"))
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

// Resolves the annotation-derived identifiers the templates render.
func (c *ecsCasting) templateData(config collectionagent.Casting) templateData {
	annotations := config.Metadata.Annotations

	// Several workloads share one cluster, so a cluster-derived name collides on
	// the second apply.
	workload := config.Metadata.Name + "-" + strings.ToLower(config.Kind().String())

	configKey := config.Spec.Collector.Kind.ConfigKey()

	return templateData{
		Casting: config,
		Region:  collectionagent.ECSRegion.Resolve(annotations),
		Cluster: Reference{Stated: collectionagent.ECSClusterARN.Resolve(annotations)},
		TaskRole: Reference{
			Stated: collectionagent.ECSTaskRoleARN.Resolve(annotations),
			Name:   workload + "-iam-task",
		},
		ExecutionRole: Reference{
			Stated: collectionagent.ECSTaskExecutionRoleARN.Resolve(annotations),
			Name:   workload + "-iam-exec",
		},
		Application: workload + "-appconfig",
		Environment: "default",
		Profile:     strings.ReplaceAll(filepath.Dir(configKey), "/", "-"),
		Source:      configKey,
		Target:      filepath.Join(configMount, filepath.Base(configKey)),
		Digest:      digest(config.Spec.Collector.Spec.Config.Data[configKey]),
		ConfigMount: configMount,
	}
}

// Resolves the identifiers the module renders. The module is imported into a
// task definition foundry never reads, so it derives everything but the role
// the ECS agent assumes.
func sidecarTemplateDataFor(config collectionagent.Casting) sidecarTemplateData {
	workload := config.Metadata.Name + "-" + strings.ToLower(config.Kind().String())

	configKey := config.Spec.Collector.Kind.ConfigKey()

	return sidecarTemplateData{
		Casting:   config,
		Container: config.Metadata.Name + "-collector-" + config.Spec.Collector.Kind.String(),
		Parameter: "/" + workload + "/" + filepath.Dir(configKey),
		Policy:    workload + "-parameter-read",
		Source:    configKey,
		Digest:    digest(config.Spec.Collector.Spec.Config.Data[configKey]),
	}
}

// Embeds the casting so .Spec and .Metadata stay reachable from templates.
type sidecarTemplateData struct {
	collectionagent.Casting

	// Container is the collector's name inside the application's task.
	Container string

	// Parameter carries the Kind, so a CollectionAgent and an Installation of
	// the same metadata.name do not collide on one account.
	Parameter string
	Policy    string

	// Source is where the pour keeps the config, relative to the module.
	Source string

	// The collector reads its config once at start, so a changed config has to
	// replace the task.
	Digest string
}

func digest(content string) string {
	sum := sha256.Sum256([]byte(content))

	return hex.EncodeToString(sum[:])
}

// Reference is one identifier, stated by an operator or created under the
// workload's own name. Exactly one side is populated.
type Reference struct {
	Stated string
	Name   string
}

func (r Reference) IsStated() bool {
	return r.Stated != ""
}

// Embeds the casting so .Spec and .Metadata stay reachable from templates.
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

	// Source is where the pour keeps the config, relative to the root; Target is
	// where the sidecar writes it in the task.
	Profile string
	Source  string
	Target  string

	// The collector reads its config once at start, so a changed config has to
	// replace the task.
	Digest string

	ConfigMount string
}
