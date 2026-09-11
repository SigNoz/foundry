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
	data := c.templateData(config)

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

const planFile = "tfplan"

func (c *ecsCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	root := filepath.Join(outputPath, p.Dir())

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
