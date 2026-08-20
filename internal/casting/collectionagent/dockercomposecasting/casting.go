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
	buf := bytes.NewBuffer(nil)
	if err := composeYAMLTemplate.Execute(buf, config); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute compose template")
	}

	p.AddYAML(buf.Bytes(), "compose.yaml")

	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *dockerComposeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	composeFile := filepath.Join(outputPath, p.Dir(), "compose.yaml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	if err := c.checkOwnership(ctx, config); err != nil {
		return err
	}

	composeCmd, err := getComposeCommand(ctx)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "docker compose not available")
	}

	args := append(composeCmd[1:], "-f", composeFile, "up", "-d")

	c.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{composeCmd[0]}, args...), " ")))

	cmd := exec.CommandContext(ctx, composeCmd[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// checkOwnership refuses to deploy over a compose project of the same name
// that belongs to a different foundry Kind. Unlabeled containers only warn:
// they are either a pre-label foundry deployment or a foreign project.
func (c *dockerComposeCasting) checkOwnership(ctx context.Context, config collectionagent.Casting) error {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "label=com.docker.compose.project="+config.Metadata.Name,
		"--format", `{{.Label "`+v1alpha1.LabelKind.Key+`"}}`).Output()
	if err != nil {
		c.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", foundryerrors.LogAttr(err))
		return nil
	}

	owners := []domain.Owner{}
	for kind := range strings.SplitSeq(strings.TrimRight(string(out), "\n"), "\n") {
		owners = append(owners, domain.Owner{v1alpha1.LabelKind.Key: kind})
	}

	ownership := domain.NewOwnership(owners...)
	self := domain.Owner{v1alpha1.LabelKind.Key: config.Kind().String()}

	if foreign, conflict := ownership.Foreign(self); conflict {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "%q already belongs to a foundry %s on this host: choose a different metadata.name or remove the existing deployment", config.Metadata.Name, foreign[v1alpha1.LabelKind.Key])
	}

	if ownership.HasUnowned() {
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
