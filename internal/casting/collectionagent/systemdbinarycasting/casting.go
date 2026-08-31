package systemdbinarycasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/systemdtooler"
)

const (
	serviceUser = "otelcol-contrib"
	configPath  = "/etc/otelcol-contrib/config.yaml"
)

type systemdBinaryCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *systemdBinaryCasting {
	return &systemdBinaryCasting{logger: logger}
}

func (c *systemdBinaryCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newSystemdBinaryMoldingEnricher(config), nil
}

func (c *systemdBinaryCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	buf := bytes.NewBuffer(nil)
	if err := serviceTemplate.Execute(buf, config); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute service template")
	}

	p.AddINI(buf.Bytes(), config.Metadata.Name+"-collector-agent.service")

	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *systemdBinaryCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	systemd, err := systemdtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	binaryPath := config.Metadata.Annotations[collectionagent.CollectorAgentBinaryPath.Key]
	if _, err := os.Stat(binaryPath); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "failed to find the collector binary at %q: download it from the OpenTelemetry Collector releases and place it there, or set the %q annotation", binaryPath, collectionagent.CollectorAgentBinaryPath.Key)
	}

	if _, err := user.Lookup(serviceUser); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "failed to find the %q user: create it with `useradd -r -s /sbin/nologin %s` before casting", serviceUser, serviceUser)
	}

	if err := writeConfig(config); err != nil {
		return err
	}

	release := systemdtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Units:   []string{filepath.Join(outputPath, p.Dir(), config.Metadata.Name+"-collector-agent.service")},
	}

	if err := systemd.Up(ctx, release); err != nil {
		return err
	}

	return systemd.Restart(ctx, release)
}

// Melt stops and disables the unit; the poured files stay, the config foundry
// wrote is removed.
func (c *systemdBinaryCasting) Melt(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	systemd, err := systemdtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	release := systemdtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Units:   []string{filepath.Join(outputPath, p.Dir(), config.Metadata.Name+"-collector-agent.service")},
	}

	if err := systemd.Down(ctx, release); err != nil {
		return err
	}

	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to remove %q", configPath)
	}

	return nil
}

func writeConfig(config collectionagent.Casting) error {
	key := config.Spec.Collector.Kind.ConfigKey()

	content, ok := config.Spec.Collector.Spec.Config.Data[key]
	if !ok {
		return foundryerrors.Newf(foundryerrors.TypeInternal, "no collector config was molded at %q", key)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create %q", filepath.Dir(configPath))
	}

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to write %q", configPath)
	}

	return nil
}
