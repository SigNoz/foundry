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
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
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

func (c *dockerComposeCasting) Forge(ctx context.Context, config collectionagent.Casting, poursPath string) ([]domain.Material, error) {
	composeBuf := bytes.NewBuffer(nil)
	if err := composeYAMLTemplate.Execute(composeBuf, config); err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute compose template")
	}
	composeMaterial, err := domain.NewYAMLMaterial(composeBuf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create compose material")
	}

	configBuf := bytes.NewBuffer(nil)
	if err := otelConfigYAMLTemplate.Execute(configBuf, config); err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute otel-collector-config template")
	}
	configMaterial, err := domain.NewYAMLMaterial(configBuf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "otel-collector-config.yaml"))
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create otel-collector-config material")
	}

	return []domain.Material{composeMaterial, configMaterial}, nil
}

func (c *dockerComposeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string) error {
	c.logger.InfoContext(ctx, "casting collectionagent via docker compose", slog.String("casting.metadata.name", config.Metadata.Name))

	composeFile := filepath.Join(outputPath, rootcasting.DeploymentDir, "compose.yaml")
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
