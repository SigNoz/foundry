package systemdbinarycasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
)

const (
	unitDir                     = "/etc/systemd/system"
	etcDir                      = "/etc"
	sysusersDir                 = "/etc/sysusers.d"
	configMode      os.FileMode = 0644
	environmentMode os.FileMode = 0600
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
	base := config.Metadata.Name + "-collector-" + config.Spec.Collector.Kind.String()

	type pour struct {
		tmpl *domain.Template
		name string
	}

	var pours []pour

	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindAgent:
		pours = []pour{
			{agentUnitTemplate, base + ".service"},
			{agentEnvironmentTemplate, base + ".conf"},
			{agentSysusersTemplate, base + ".sysusers.conf"},
		}
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	for key, value := range config.Spec.Collector.Spec.Env {
		if strings.ContainsAny(value, "\n\r") {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to forge the environment file: the value of %q spans more than one line, which an EnvironmentFile cannot carry", key)
		}
	}

	dir := filepath.Dir(config.Spec.Collector.Kind.ConfigKey())

	for _, pour := range pours {
		buf := bytes.NewBuffer(nil)
		if err := pour.tmpl.Execute(buf, config); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", pour.tmpl.Name())
		}

		switch pour.tmpl.Format().String() {
		case domain.FormatINI.String():
			p.AddINI(buf.Bytes(), dir, pour.name)
		case domain.FormatText.String():
			p.AddBlob(buf.Bytes(), dir, pour.name)
		default:
			p.AddYAML(buf.Bytes(), dir, pour.name)
		}
	}

	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *systemdBinaryCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	if config.Spec.Collector.Kind != collectionagent.CollectorKindAgent {
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	c.logger.InfoContext(ctx, "Installing the collector service",
		slog.String("release", config.Metadata.Name),
	)

	binaryPath := collectionagent.CollectorAgentBinaryPath.Resolve(config.Metadata.Annotations)
	if _, err := os.Stat(binaryPath); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "failed to find the collector binary at %q: download it from the OpenTelemetry Collector releases and place it there, or set the %q annotation", binaryPath, collectionagent.CollectorAgentBinaryPath.Key)
	}

	base := config.Metadata.Name + "-collector-" + config.Spec.Collector.Kind.String()
	etc := filepath.Join(etcDir, base)
	if err := os.MkdirAll(etc, 0755); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create %q", etc)
	}

	// Stock distributions ship /usr/lib/sysusers.d but not /etc/sysusers.d.
	if err := os.MkdirAll(sysusersDir, 0755); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create %q", sysusersDir)
	}

	poursDir := filepath.Join(outputPath, p.Dir(), filepath.Dir(config.Spec.Collector.Kind.ConfigKey()))
	configName := config.Spec.Collector.Kind.String() + ".yaml"

	if err := c.provision(filepath.Join(poursDir, configName), filepath.Join(etc, configName), configMode); err != nil {
		return err
	}

	if err := c.provision(filepath.Join(poursDir, base+".conf"), filepath.Join(etc, base+".conf"), environmentMode); err != nil {
		return err
	}

	if err := c.provision(filepath.Join(poursDir, base+".service"), filepath.Join(unitDir, base+".service"), configMode); err != nil {
		return err
	}

	if err := c.provision(filepath.Join(poursDir, base+".sysusers.conf"), filepath.Join(sysusersDir, base+".conf"), configMode); err != nil {
		return err
	}

	// systemd-sysusers.service skips itself on ConditionNeedsUpdate.
	if err := c.run(ctx, "systemd-sysusers", filepath.Join(sysusersDir, base+".conf")); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create the service user %s", base)
	}

	if err := c.run(ctx, "systemctl", "daemon-reload"); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "systemd daemon-reload failed")
	}

	if err := c.run(ctx, "systemctl", "enable", base+".service"); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to enable unit %s", base+".service")
	}

	if err := c.run(ctx, "systemctl", "restart", base+".service"); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to restart unit %s", base+".service")
	}

	c.logger.InfoContext(ctx, "Collector service installed successfully")

	return nil
}

// The mode is set after the write: WriteFile masks it with the umask and ignores it on an existing file.
func (c *systemdBinaryCasting) provision(src string, dst string, mode os.FileMode) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to read %q: forge before casting", src)
	}

	if err := os.WriteFile(dst, content, mode); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to write %q", dst)
	}

	if err := os.Chmod(dst, mode); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to set the mode on %q", dst)
	}

	return nil
}

func (c *systemdBinaryCasting) run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
