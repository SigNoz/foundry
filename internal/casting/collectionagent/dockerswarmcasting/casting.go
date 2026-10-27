package dockerswarmcasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

type dockerSwarmCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *dockerSwarmCasting {
	return &dockerSwarmCasting{logger: logger}
}

func (c *dockerSwarmCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newDockerSwarmMoldingEnricher(), nil
}

func (c *dockerSwarmCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	var tmpl *domain.Template

	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindAgent:
		tmpl = composeYAMLTemplate
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, config); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute compose template")
	}

	p.AddYAML(buf.Bytes(), "compose.yaml")

	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *dockerSwarmCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	composeFile := filepath.Join(outputPath, p.Dir(), "compose.yaml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	return c.run(ctx, config, "deploy", "-d", "-c", composeFile, config.Metadata.Name)
}

func (c *dockerSwarmCasting) Down(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	composeFile := filepath.Join(outputPath, p.Dir(), "compose.yaml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	return c.run(ctx, config, "rm", config.Metadata.Name)
}

func (c *dockerSwarmCasting) run(ctx context.Context, config collectionagent.Casting, args ...string) error {
	if err := c.checkOwnership(ctx, config); err != nil {
		return err
	}

	argv := append([]string{"stack"}, args...)

	c.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{"docker"}, argv...), " ")))

	cmd := exec.CommandContext(ctx, "docker", argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run docker stack %s", args[0])
	}

	return nil
}

// checkOwnership refuses to deploy over a swarm stack of the same name that
// belongs to a different foundry Kind. Unlabeled task containers only warn:
// they are either a pre-label foundry deployment or a foreign stack.
func (c *dockerSwarmCasting) checkOwnership(ctx context.Context, config collectionagent.Casting) error {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "label=com.docker.stack.namespace="+config.Metadata.Name,
		"--format", `{{.Label "`+v1alpha1.LabelKind.Key+`"}}`).Output()
	if err != nil {
		c.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", foundryerrors.LogAttr(err))
		return nil
	}

	ownership := domain.ParseOwnership(string(out))

	if foreign, conflict := ownership.Foreign(config.Kind().String()); conflict {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "%q already belongs to a foundry %s on this host: choose a different metadata.name or remove the existing deployment", config.Metadata.Name, foreign)
	}

	if ownership.HasUnlabeled() {
		c.logger.WarnContext(ctx, "swarm stack has task containers without foundry ownership labels", slog.String("stack", config.Metadata.Name))
	}

	return nil
}
