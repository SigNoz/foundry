package systemddebcasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
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
	unitName   = "otelcol-contrib.service"
	configPath = "/etc/otelcol-contrib/config.yaml"
	dropInDir  = "otelcol-contrib.service.d"
	dropInName = "foundry.conf"
	dropInPath = "/etc/systemd/system/" + dropInDir + "/" + dropInName
)

type systemdDebCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *systemdDebCasting {
	return &systemdDebCasting{logger: logger}
}

func (c *systemdDebCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newSystemdDebMoldingEnricher(), nil
}

func (c *systemdDebCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	buf := bytes.NewBuffer(nil)
	if err := dropInTemplate.Execute(buf, config); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute drop-in template")
	}

	p.AddINI(buf.Bytes(), dropInDir, dropInName)

	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *systemdDebCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	systemd, err := systemdtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	if _, err := systemd.Cat(ctx, systemdtooler.Options{Unit: unitName}); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "failed to find the %q unit: install the package first with `sudo dpkg -i otelcol-contrib_<version>_linux_<arch>.deb`", unitName)
	}

	if err := writeConfig(config); err != nil {
		return err
	}

	if err := installDropIn(outputPath, p); err != nil {
		return err
	}

	release := systemdtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Units:   []string{unitName},
	}

	if err := systemd.Up(ctx, release); err != nil {
		return err
	}

	return systemd.Restart(ctx, release)
}

// Melt removes the drop-in before stopping the unit, so its daemon-reload
// applies the removal; the package-owned unit and config.yaml (a package
// conffile foundry never recorded) stay.
func (c *systemdDebCasting) Melt(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	systemd, err := systemdtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	if err := os.Remove(dropInPath); err != nil && !os.IsNotExist(err) {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to remove %q", dropInPath)
	}

	release := systemdtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Units:   []string{unitName},
	}

	return systemd.Down(ctx, release)
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

func installDropIn(outputPath string, p *pourer.Pourer) error {
	content, err := os.ReadFile(filepath.Join(outputPath, p.Dir(), dropInDir, dropInName))
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to read the poured drop-in")
	}

	if err := os.MkdirAll(filepath.Dir(dropInPath), 0755); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create %q", filepath.Dir(dropInPath))
	}

	if err := os.WriteFile(dropInPath, content, 0644); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to write %q", dropInPath)
	}

	return nil
}
