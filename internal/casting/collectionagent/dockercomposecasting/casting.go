package dockercomposecasting

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

type dockerComposeCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *dockerComposeCasting {
	return &dockerComposeCasting{logger: logger}
}

func (c *dockerComposeCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newDockerComposeMoldingEnricher(), nil
}

func (c *dockerComposeCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
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

func (c *dockerComposeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	c.logger.InfoContext(ctx, "deploying the compose project", slog.String("project", config.Metadata.Name))

	composeFile := filepath.Join(outputPath, p.Dir(), "compose.yaml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	return c.docker(ctx, config, composeFile, "up", "-d")
}

func (c *dockerComposeCasting) Down(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	c.logger.InfoContext(ctx, "removing the compose project", slog.String("project", config.Metadata.Name))

	composeFile := filepath.Join(outputPath, p.Dir(), "compose.yaml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	return c.docker(ctx, config, composeFile, "down")
}

func (c *dockerComposeCasting) docker(ctx context.Context, config collectionagent.Casting, composeFile string, args ...string) error {
	composeCmd, err := getComposeCommand(ctx)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "docker compose not available")
	}

	if err := c.checkOwnership(ctx, config, composeCmd, composeFile); err != nil {
		return err
	}

	argv := append(composeCmd[1:], "-f", composeFile)
	argv = append(argv, args...)

	c.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{composeCmd[0]}, argv...), " ")))

	cmd := exec.CommandContext(ctx, composeCmd[0], argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run docker compose %s", args[0])
	}

	return nil
}

// checkOwnership refuses a compose project of the same name that belongs to a different foundry Kind; unlabeled containers only warn.
func (c *dockerComposeCasting) checkOwnership(ctx context.Context, config collectionagent.Casting, composeCmd []string, composeFile string) error {
	argv := append(composeCmd[1:], "-f", composeFile, "ps", "-a", "-q")

	out, err := exec.CommandContext(ctx, composeCmd[0], argv...).Output()
	if err != nil {
		c.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", foundryerrors.LogAttr(err))
		return nil
	}

	ids := strings.Fields(string(out))

	if len(ids) == 0 {
		return nil
	}

	argv = append([]string{"inspect", "--format", `{{ index .Config.Labels "` + v1alpha1.LabelKind.Key + `" }}`}, ids...)

	out, err = exec.CommandContext(ctx, "docker", argv...).Output()
	if err != nil {
		c.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", foundryerrors.LogAttr(err))
		return nil
	}

	ownership := domain.ParseOwnership(string(out))

	if foreign, conflict := ownership.Foreign(config.Kind().String()); conflict {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "%q already belongs to a foundry %s on this host: choose a different metadata.name or remove the existing deployment", config.Metadata.Name, foreign)
	}

	if ownership.HasUnlabeled() {
		c.logger.WarnContext(ctx, "compose project has containers without foundry ownership labels", slog.String("project", config.Metadata.Name))
	}

	return nil
}

func getComposeCommand(ctx context.Context) ([]string, error) {
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.CommandContext(ctx, "docker", "compose", "version")

		if err := cmd.Run(); err == nil {
			return []string{"docker", "compose"}, nil
		}
	}

	if _, err := exec.LookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}, nil
	}

	return nil, foundryerrors.Newf(foundryerrors.TypeNotFound, "neither 'docker compose' nor 'docker-compose' is available")
}
