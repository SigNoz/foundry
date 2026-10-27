package awskubernetesserverlesskustomizecasting

import (
	"bytes"
	"context"
	"fmt"
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

type awsKubernetesServerlessKustomizeCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *awsKubernetesServerlessKustomizeCasting {
	return &awsKubernetesServerlessKustomizeCasting{logger: logger}
}

func (c *awsKubernetesServerlessKustomizeCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newAwsKubernetesServerlessKustomizeMoldingEnricher(), nil
}

func (c *awsKubernetesServerlessKustomizeCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	data := templateDataFor(config)

	var tmpls, workload, collector []*domain.Template

	// Fargate schedules no DaemonSet, so the deployment collects for the cluster
	// and the sidecar runs inside the application's pods.
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindDeployment:
		tmpls = []*domain.Template{
			kustomizationTemplate,
			namespaceTemplate,
			deploymentServiceaccountTemplate,
			deploymentClusterroleTemplate,
			deploymentClusterrolebindingTemplate,
		}
		workload = []*domain.Template{deploymentServiceTemplate, deploymentTemplate}
		collector = []*domain.Template{deploymentOpenTelemetryCollectorTemplate}
	case collectionagent.CollectorKindSidecar:
		body := bytes.NewBuffer(nil)
		if err := sidecarPodtemplateTemplate.Execute(body, data); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", sidecarPodtemplateTemplate.Name())
		}

		data.PodTemplate = body.String()

		tmpls = []*domain.Template{sidecarKustomizationTemplate, sidecarRoleTemplate, sidecarRolebindingTemplate}
		workload = []*domain.Template{sidecarPatchTemplate, sidecarPatchCronjobTemplate}
		collector = []*domain.Template{sidecarOpenTelemetryCollectorTemplate}
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	// The kind's directory is a kustomize root of its own, so a casting file
	// holding several collection agents pours a tree per document.
	dir := filepath.Dir(config.Spec.Collector.Kind.ConfigKey())

	switch data.Controller {
	case collectionagent.CollectorControllerDefault:
		tmpls = append(tmpls, workload...)
	case collectionagent.CollectorControllerOpenTelemetryOperator:
		tmpls = append(tmpls, collector...)
	default:
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "collector controller %q is not supported, state %q or %q", data.Controller, collectionagent.CollectorControllerDefault, collectionagent.CollectorControllerOpenTelemetryOperator)
	}

	for _, tmpl := range tmpls {
		buf := bytes.NewBuffer(nil)
		if err := tmpl.Execute(buf, data); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", tmpl.Name())
		}

		p.AddYAML(buf.Bytes(), dir, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
	}

	// The collector config, inside the kustomize root so the configMapGenerator
	// reaches it by relative path.
	if data.Controller == collectionagent.CollectorControllerDefault {
		for path, content := range config.Spec.Collector.Spec.Config.Data {
			p.AddYAML([]byte(content), path)
		}
	}

	return nil
}

func (c *awsKubernetesServerlessKustomizeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	kustomizeDir := filepath.Join(outputPath, p.Dir(), filepath.Dir(config.Spec.Collector.Kind.ConfigKey()))

	if config.Spec.Collector.Kind != collectionagent.CollectorKindSidecar {
		return c.apply(ctx, config, kustomizeDir)
	}

	if collectionagent.KubernetesCollectorController.Resolve(config.Metadata.Annotations) == collectionagent.CollectorControllerOpenTelemetryOperator {
		c.logger.InfoContext(ctx, "The sidecar pour is a kustomize base; list it under resources in your application's kustomization, add the annotation to each workload's pod template, then apply the application and restart the workloads after a config change.",
			slog.String("base", kustomizeDir),
			slog.String("annotation", fmt.Sprintf("sidecar.opentelemetry.io/inject: %q", config.Metadata.Name+"-collector-sidecar")))

		return nil
	}

	c.logger.InfoContext(ctx, "The sidecar pour is a kustomize base; list it under resources in your application's kustomization and its patch.yaml under patches with a target naming each workload (patch-cronjob.yaml for a CronJob), or copy its initContainers and volumes entries into workloads managed another way, then apply the application and restart the workloads after a config change.",
		slog.String("base", kustomizeDir),
		slog.String("patch", filepath.Join(kustomizeDir, "patch.yaml")))

	return nil
}

func (c *awsKubernetesServerlessKustomizeCasting) Down(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	return foundryerrors.Newf(foundryerrors.TypeUnsupported, "down is not supported for %s %q", config.Kind(), config.Metadata.Name)
}

func (c *awsKubernetesServerlessKustomizeCasting) apply(ctx context.Context, config collectionagent.Casting, kustomizeDir string) error {
	c.logger.InfoContext(ctx, "Applying kustomize manifests",
		slog.String("release", config.Metadata.Name),
		slog.String("namespace", namespace(config)),
	)

	if _, err := os.Stat(filepath.Join(kustomizeDir, "kustomization.yaml")); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "kustomization.yaml does not exist at path: %s, run 'forge' first", kustomizeDir)
	}

	if err := c.kubectl(ctx, "apply", "-k", kustomizeDir); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "kubectl apply -k failed")
	}

	c.logger.InfoContext(ctx, "Kustomize manifests applied successfully")

	return nil
}

// The embedded casting keeps $.Spec and $.Metadata reachable from the templates.
type templateData struct {
	collectionagent.Casting

	Namespace       string
	StatedNamespace string
	Controller      string
	PodTemplate     string
}

func templateDataFor(config collectionagent.Casting) templateData {
	return templateData{
		Casting:         config,
		Namespace:       namespace(config),
		StatedNamespace: collectionagent.KubernetesNamespace.Resolve(config.Metadata.Annotations),
		Controller:      collectionagent.KubernetesCollectorController.Resolve(config.Metadata.Annotations),
	}
}

// The annotation's default cannot name metadata.name, so the fallback lives here.
func namespace(config collectionagent.Casting) string {
	ns := collectionagent.KubernetesNamespace.Resolve(config.Metadata.Annotations)
	if ns == "" {
		ns = config.Metadata.Name
	}

	return ns
}

func (c *awsKubernetesServerlessKustomizeCasting) kubectl(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	c.logger.DebugContext(ctx, "Running command",
		slog.String("command", fmt.Sprintf("kubectl %s", strings.Join(args, " "))))

	if err := cmd.Run(); err != nil {
		c.logger.ErrorContext(ctx, "kubectl failed", foundryerrors.LogAttr(err))

		return err
	}

	return nil
}
