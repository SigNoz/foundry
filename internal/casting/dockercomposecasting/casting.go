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
	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ rootcasting.Casting = (*dockerComposeCasting)(nil)

type dockerComposeCasting struct {
	logger   *slog.Logger
	castings []*domain.Template
}

func New(logger *slog.Logger) *dockerComposeCasting {
	return &dockerComposeCasting{
		logger: logger,
		castings: []*domain.Template{
			composeYAMLTemplate,
		},
	}
}

func (casting *dockerComposeCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newDockerComposeMoldingEnricher(casting.logger, config)
}

func (casting *dockerComposeCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute compose yaml template")
	}

	composeMaterial, err := domain.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create compose yaml material")
	}

	materials := []domain.Material{composeMaterial}

	// Add telemetrykeeper config files
	for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create telemetrykeeper config material")
		}
		materials = append(materials, material)
	}

	// Add telemetrystore config files
	for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create telemetrystore config material")
		}
		materials = append(materials, material)
	}

	// Add metastore config files
	for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "metastore", config.Spec.MetaStore.Kind.String(), filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create metastore config material")
		}
		materials = append(materials, material)
	}

	// Add signoz config files
	for filename, content := range config.Spec.Signoz.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "signoz", filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create signoz config material")
		}
		materials = append(materials, material)
	}

	// Add ingester config files
	for filename, content := range config.Spec.Ingester.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "ingester", filename))
		if err != nil {
			return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to create ingester config material")
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func (casting *dockerComposeCasting) Cast(ctx context.Context, config installation.Casting, outputPath string) error {
	casting.logger.InfoContext(ctx, "deploying the compose project", slog.String("project", config.Metadata.Name))

	composeFile := filepath.Join(outputPath, rootcasting.DeploymentDir, "compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	if _, err := casting.checkOwnership(ctx, config); err != nil {
		return err
	}

	return casting.docker(ctx, config, composeFile, "up", "-d")
}

func (casting *dockerComposeCasting) Down(ctx context.Context, config installation.Casting, outputPath string) error {
	casting.logger.InfoContext(ctx, "removing the compose project", slog.String("project", config.Metadata.Name))

	found, err := casting.checkOwnership(ctx, config)
	if err != nil {
		return err
	}

	if !found {
		casting.logger.WarnContext(ctx, "nothing deployed under this name, nothing to take down", slog.String("project", config.Metadata.Name))
		return nil
	}

	composeFile := filepath.Join(outputPath, rootcasting.DeploymentDir, "compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	return casting.docker(ctx, config, composeFile, "down")
}

func (casting *dockerComposeCasting) docker(ctx context.Context, config installation.Casting, composeFile string, args ...string) error {
	composeCmd, err := getComposeCommand(ctx)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "docker compose not available")
	}

	argv := append(composeCmd[1:], "-f", composeFile)
	argv = append(argv, args...)

	casting.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{composeCmd[0]}, argv...), " ")))

	cmd := exec.CommandContext(ctx, composeCmd[0], argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run docker compose %s", args[0])
	}

	return nil
}

// checkOwnership reports whether anything is deployed under the project name and refuses a project that belongs to a different foundry Kind; unlabeled containers only warn.
func (casting *dockerComposeCasting) checkOwnership(ctx context.Context, config installation.Casting) (bool, error) {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a", "-q", "--filter", "label=com.docker.compose.project="+config.Metadata.Name).Output()
	if err != nil {
		casting.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", foundryerrors.LogAttr(err))
		return true, nil
	}

	ids := strings.Fields(string(out))

	if len(ids) == 0 {
		return false, nil
	}

	argv := append([]string{"inspect", "--format", `{{ index .Config.Labels "` + v1alpha1.LabelKind.Key + `" }}`}, ids...)

	out, err = exec.CommandContext(ctx, "docker", argv...).Output()
	if err != nil {
		casting.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", foundryerrors.LogAttr(err))
		return true, nil
	}

	ownership := domain.ParseOwnership(string(out))

	if foreign, conflict := ownership.Foreign(config.Kind().String()); conflict {
		return false, foundryerrors.Newf(foundryerrors.TypeInvalidInput, "%q already belongs to a foundry %s on this host: choose a different metadata.name or remove the existing deployment", config.Metadata.Name, foreign)
	}

	if ownership.HasUnlabeled() {
		casting.logger.WarnContext(ctx, "compose project has containers without foundry ownership labels", slog.String("project", config.Metadata.Name))
	}

	return true, nil
}

func getComposeMaterial(config *installation.Casting, path string) (domain.StructuredMaterial, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute compose yaml template")
	}

	return domain.NewYAMLMaterial(buf.Bytes(), path)
}

// getComposeCommand detects the available docker compose command.
// It checks for "docker compose" (newer, preferred) first, then falls back to "docker-compose" (legacy).
func getComposeCommand(ctx context.Context) ([]string, error) {
	// Check "docker compose" first (newer, preferred)
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.CommandContext(ctx, "docker", "compose", "version")
		if err := cmd.Run(); err == nil {
			return []string{"docker", "compose"}, nil
		}
	}

	// Fallback to "docker-compose" (legacy)
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}, nil
	}

	return nil, foundryerrors.Newf(foundryerrors.TypeNotFound, "neither 'docker compose' nor 'docker-compose' is available")
}
