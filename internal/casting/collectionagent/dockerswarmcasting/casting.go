package dockerswarmcasting

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/dockerswarmtooler"
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

func (c *dockerSwarmCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	swarm, err := dockerswarmtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := dockerswarmtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		File:    filepath.Join(outputPath, p.Dir(), strings.TrimSuffix(composeYAMLTemplate.Name(), ".gotmpl")),
	}

	return swarm.Up(ctx, release)
}

// Uncast removes the stack's services and networks; the volumes holding
// component data stay.
func (c *dockerSwarmCasting) Uncast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	swarm, err := dockerswarmtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := dockerswarmtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		File:    filepath.Join(outputPath, p.Dir(), strings.TrimSuffix(composeYAMLTemplate.Name(), ".gotmpl")),
	}

	return swarm.Down(ctx, release)
}
