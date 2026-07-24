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

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
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

func (c *dockerSwarmCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	composeFile := filepath.Join(outputPath, p.Dir(), "compose.yaml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
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
