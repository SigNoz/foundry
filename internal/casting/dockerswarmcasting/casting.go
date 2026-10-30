package dockerswarmcasting

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
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ rootcasting.Casting = (*dockerSwarmCasting)(nil)

type dockerSwarmCasting struct {
	logger   *slog.Logger
	castings []*domain.Template
}

func New(logger *slog.Logger) *dockerSwarmCasting {
	return &dockerSwarmCasting{
		logger: logger,
		castings: []*domain.Template{
			composeYAMLTemplate,
		},
	}
}

func (casting *dockerSwarmCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newDockerSwarmMoldingEnricher(config)
}

func (casting *dockerSwarmCasting) Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error) {

	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to execute compose yaml template")
	}

	composeMaterial, err := domain.NewYAMLMaterial(buf.Bytes(), filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create compose yaml material")
	}

	materials := []domain.Material{composeMaterial}

	for filename, content := range config.Spec.TelemetryKeeper.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String(), filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create telemetrykeeper config material")
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.TelemetryStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String(), filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create telemetrystore config material")
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.MetaStore.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "metastore", config.Spec.MetaStore.Kind.String(), filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create metastore config material")
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.Signoz.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "signoz", filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create signoz config material")
		}
		materials = append(materials, material)
	}

	for filename, content := range config.Spec.Ingester.Spec.Config.Data {
		material, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, "ingester", filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create ingester config material")
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func (casting *dockerSwarmCasting) Cast(ctx context.Context, config installation.Casting, outputPath string) error {
	casting.logger.InfoContext(ctx, "Deploying stack to Docker Swarm")

	composeFile := filepath.Join(outputPath, rootcasting.DeploymentDir, "compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return errors.Newf(errors.TypeNotFound, "compose file does not exist at path: %s", composeFile)
	}

	ids, err := casting.capture(ctx, "ls", "-q", "--filter", "label=com.docker.stack.namespace="+config.Metadata.Name)

	var labels []byte
	if services := strings.Fields(string(ids)); len(services) > 0 {
		labels, err = casting.capture(ctx, append([]string{"inspect", "--format", `{{ index .Spec.TaskTemplate.ContainerSpec.Labels "` + v1alpha1.LabelKind.Key + `" }}`}, services...)...)
	}

	if err != nil {
		casting.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", errors.LogAttr(err))
	}

	ownership := domain.ParseOwnership(string(labels))

	if foreign, conflict := ownership.Foreign(config.Kind().String()); conflict {
		return errors.Newf(errors.TypeInvalidInput, "%q already belongs to a foundry %s on this host: choose a different metadata.name or remove the existing deployment", config.Metadata.Name, foreign)
	}

	if ownership.HasUnlabeled() {
		casting.logger.WarnContext(ctx, "swarm stack has services without foundry ownership labels", slog.String("stack", config.Metadata.Name))
	}

	err = casting.docker(ctx, "deploy", "-d", "-c", composeFile, config.Metadata.Name)
	if err != nil {
		casting.logger.ErrorContext(ctx, "Stack deploy failed", slog.String("error", err.Error()))
		return err
	}

	casting.logger.InfoContext(ctx, "Stack deployed successfully")

	return nil
}

func (casting *dockerSwarmCasting) Down(ctx context.Context, config installation.Casting, outputPath string) error {
	casting.logger.InfoContext(ctx, "removing the swarm stack", slog.String("stack", config.Metadata.Name))

	ids, err := casting.capture(ctx, "ls", "-q", "--filter", "label=com.docker.stack.namespace="+config.Metadata.Name)

	var labels []byte
	if services := strings.Fields(string(ids)); len(services) > 0 {
		labels, err = casting.capture(ctx, append([]string{"inspect", "--format", `{{ index .Spec.TaskTemplate.ContainerSpec.Labels "` + v1alpha1.LabelKind.Key + `" }}`}, services...)...)
	}

	if err != nil {
		return err
	}

	ownership := domain.ParseOwnership(string(labels))

	if foreign, conflict := ownership.Foreign(config.Kind().String()); conflict {
		return errors.Newf(errors.TypeInvalidInput, "%q already belongs to a foundry %s on this host: choose a different metadata.name or remove the existing deployment", config.Metadata.Name, foreign)
	}

	if ownership.HasUnlabeled() {
		return errors.Newf(errors.TypeInvalidInput, "%q has services without foundry ownership labels: cast once to stamp them, then cast down", config.Metadata.Name)
	}

	return casting.docker(ctx, "rm", config.Metadata.Name)
}

func (casting *dockerSwarmCasting) docker(ctx context.Context, args ...string) error {
	argv := append([]string{"stack"}, args...)

	casting.logger.DebugContext(ctx, "running command", slog.String("command", "docker "+strings.Join(argv, " ")))

	cmd := exec.CommandContext(ctx, "docker", argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to run docker stack %s", args[0])
	}

	return nil
}

func (casting *dockerSwarmCasting) capture(ctx context.Context, args ...string) ([]byte, error) {
	argv := append([]string{"service"}, args...)

	casting.logger.DebugContext(ctx, "running command", slog.String("command", "docker "+strings.Join(argv, " ")))

	cmd := exec.CommandContext(ctx, "docker", argv...)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to run docker service %s: start the docker daemon on a swarm manager", args[0])
	}

	return out, nil
}

func getComposeMaterial(config *installation.Casting, path string) (domain.StructuredMaterial, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to execute compose yaml template")
	}

	return domain.NewYAMLMaterial(buf.Bytes(), path)
}
