package dockercomposecasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
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

	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	composeCmd, err := getComposeCommand(runctx)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "docker compose not available")
	}

	args := append(composeCmd[1:], "-f", composeFile, "up", "-d")

	c.logger.DebugContext(runctx, "running command", slog.String("command", strings.Join(append([]string{composeCmd[0]}, args...), " ")))

	cmd := exec.CommandContext(runctx, composeCmd[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
