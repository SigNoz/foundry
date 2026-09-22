package ecsfargateterraformcasting

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

type ecsFargateTerraformCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *ecsFargateTerraformCasting {
	return &ecsFargateTerraformCasting{logger: logger}
}

func (c *ecsFargateTerraformCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newEcsFargateMoldingEnricher(), nil
}

func (c *ecsFargateTerraformCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	var tmpls []*domain.Template

	// A fargate task has no container instance, so the collector runs inside the
	// application's own task and no other kind has anything to run on.
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindSidecar:
		tmpls = []*domain.Template{versionsTF, mainTF, outputsTF}
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	// The kind's directory is a terraform module of its own, so a casting file
	// holding several collection agents pours a module per document.
	dir := filepath.Dir(config.Spec.Collector.Kind.ConfigKey())

	data := templateDataFor(config)

	for _, tmpl := range tmpls {
		material, err := tmpl.Render(data, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
		if err != nil {
			return err
		}

		p.AddJSON(material.FmtContents(), dir, material.Path())
	}

	// Terraform reads the config off disk at plan time, so the pour is the
	// source the delivered configuration is built from.
	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *ecsFargateTerraformCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	c.logger.InfoContext(ctx, "The sidecar is a terraform module; import it from your task definition's terraform.",
		slog.String("module", filepath.Join(outputPath, p.Dir(), filepath.Dir(config.Spec.Collector.Kind.ConfigKey()))))

	return nil
}

// Resolves the identifiers the module renders. The module is imported into a
// task definition foundry never reads, so it derives every one of them.
func templateDataFor(config collectionagent.Casting) templateData {
	workload := config.Metadata.Name + "-" + strings.ToLower(config.Kind().String())

	configKey := config.Spec.Collector.Kind.ConfigKey()

	return templateData{
		Casting:   config,
		Container: config.Metadata.Name + "-collector-" + config.Spec.Collector.Kind.String(),
		ExecutionRole: Reference{
			Stated: collectionagent.ECSTaskExecutionRoleARN.Resolve(config.Metadata.Annotations),
			Name:   workload + "-iam-exec",
		},
		Parameter: workload + "-ssm-" + strings.ReplaceAll(filepath.Dir(configKey), "/", "-"),
		Policy:    workload + "-iam-exec-ssm-read",
		Source:    filepath.Base(configKey),
	}
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

	// Container is the collector's name inside the application's task.
	Container string

	// ExecutionRole is the task definition's own: the ECS agent resolves the
	// config secret with it, so a stated one is adopted rather than replaced.
	ExecutionRole Reference

	// Parameter and Policy carry the Kind, so a CollectionAgent and an
	// Installation of the same metadata.name do not collide on one account.
	Parameter string
	Policy    string

	// Source is the config file the pour leaves beside the module.
	Source string
}
