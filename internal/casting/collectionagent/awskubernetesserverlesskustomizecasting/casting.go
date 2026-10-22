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

	var tmpls, workload, collector, component []*domain.Template

	// Fargate schedules no DaemonSet, so the deployment collects for the cluster
	// and the sidecar joins the application's pods through a kustomize component.
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
		collector = []*domain.Template{sidecarKustomizationTemplate, namespaceTemplate, sidecarOpenTelemetryCollectorTemplate}
		component = []*domain.Template{
			sidecarComponentTemplate,
			sidecarKustomizeconfigTemplate,
			sidecarRoleTemplate,
			sidecarRolebindingTemplate,
			sidecarPatchTemplateTemplate,
			sidecarPatchCronjobTemplate,
		}
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	// The kind's directory is a kustomize root of its own, so a casting file
	// holding several collection agents pours a tree per document.
	dir := filepath.Dir(config.Spec.Collector.Kind.ConfigKey())
	componentDir := dir

	switch data.Controller {
	case collectionagent.CollectorControllerDefault:
		tmpls = append(tmpls, workload...)
	case collectionagent.CollectorControllerOpenTelemetryOperator:
		tmpls = append(tmpls, collector...)
		// The root applies the operator's resource, so the component sits in a
		// directory the root does not reference.
		componentDir = filepath.Join(dir, "component")
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

	if len(component) > 0 {
		body := bytes.NewBuffer(nil)
		if err := sidecarPodtemplateTemplate.Execute(body, data); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", sidecarPodtemplateTemplate.Name())
		}

		withBody := componentData{templateData: data, PodTemplate: body.String()}

		for _, tmpl := range component {
			buf := bytes.NewBuffer(nil)
			if err := tmpl.Execute(buf, withBody); err != nil {
				return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", tmpl.Name())
			}

			p.AddYAML(buf.Bytes(), componentDir, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
		}
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

	// Under the default controller the sidecar is the component alone, which
	// the application's own kustomization applies.
	component := kustomizeDir
	if collectionagent.KubernetesCollectorController.Resolve(config.Metadata.Annotations) == collectionagent.CollectorControllerOpenTelemetryOperator {
		if err := c.apply(ctx, config, kustomizeDir); err != nil {
			return err
		}

		component = filepath.Join(kustomizeDir, "component")
	}

	c.logger.InfoContext(ctx, "The sidecar is a kustomize component; list it under components in your application's kustomization and label each workload foundry.signoz.io/kubernetes-collector-sidecar: enabled.",
		slog.String("component", component))

	return nil
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

	Namespace  string
	Controller string
}

// The component's patches root one pod template body at each workload kind's template.
type componentData struct {
	templateData

	PodTemplate string
}

func templateDataFor(config collectionagent.Casting) templateData {
	return templateData{
		Casting:    config,
		Namespace:  namespace(config),
		Controller: collectionagent.KubernetesCollectorController.Resolve(config.Metadata.Annotations),
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
