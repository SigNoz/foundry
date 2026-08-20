package dockercomposecasting

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/signoz/foundry/internal/runner"
	"github.com/signoz/foundry/internal/runner/composerunner"
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

func (c *dockerComposeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, runners []runner.Runner) error {
	compose, err := composerunner.Lookup(runners)
	if err != nil {
		return err
	}

	return compose.Up(ctx, c.options(config, outputPath, p))
}

// Uncast removes the agent's containers and networks; volumes stay.
func (c *dockerComposeCasting) Uncast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, runners []runner.Runner) error {
	compose, err := composerunner.Lookup(runners)
	if err != nil {
		return err
	}

	return compose.Down(ctx, c.options(config, outputPath, p))
}

// options states the project this casting owns, so the runner refuses a
// project of the same name that belongs to another foundry Kind.
func (c *dockerComposeCasting) options(config collectionagent.Casting, outputPath string, p *pourer.Pourer) composerunner.Options {
	return composerunner.Options{
		File:    filepath.Join(outputPath, p.Dir(), "compose.yaml"),
		Project: config.Metadata.Name,
		Owner:   config.Labels(),
	}
}
