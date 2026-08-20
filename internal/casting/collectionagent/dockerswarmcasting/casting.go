package dockerswarmcasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/signoz/foundry/internal/runner"
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

func (c *dockerSwarmCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, _ []runner.Runner) error {
	composeFile := filepath.Join(outputPath, p.Dir(), "compose.yaml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	if err := c.checkOwnership(ctx, config); err != nil {
		return err
	}

	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := []string{"stack", "deploy", "-d", "-c", composeFile, config.Metadata.Name}

	c.logger.DebugContext(runctx, "running command", slog.String("command", strings.Join(append([]string{"docker"}, args...), " ")))

	cmd := exec.CommandContext(runctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Uncast is not implemented for this casting yet.
func (c *dockerSwarmCasting) Uncast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, _ []runner.Runner) error {
	return foundryerrors.Newf(foundryerrors.TypeUnsupported, "uncast is not implemented for this casting yet")
}

// checkOwnership refuses to deploy over a swarm stack of the same name that
// belongs to a different foundry Kind. Task containers that record no owner
// only warn: they are either a pre-label foundry deployment or a foreign
// stack.
func (c *dockerSwarmCasting) checkOwnership(ctx context.Context, config collectionagent.Casting) error {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "label=com.docker.stack.namespace="+config.Metadata.Name,
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
		c.logger.WarnContext(ctx, "swarm stack has task containers without foundry ownership labels", slog.String("stack", config.Metadata.Name))
	}

	return nil
}
