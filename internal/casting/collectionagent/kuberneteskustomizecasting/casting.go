package kuberneteskustomizecasting

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

type kubernetesKustomizeCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *kubernetesKustomizeCasting {
	return &kubernetesKustomizeCasting{logger: logger}
}

func (c *kubernetesKustomizeCasting) Enricher(ctx context.Context, config *collectionagent.Casting) (collectionagentmolding.MoldingEnricher, error) {
	return newKubernetesKustomizeMoldingEnricher(), nil
}

func (c *kubernetesKustomizeCasting) Forge(ctx context.Context, config collectionagent.Casting, p *pourer.Pourer) error {
	data := templateDataFor(config)

	tmpls := []*domain.Template{kustomizationTemplate, namespaceTemplate}

	var workload []*domain.Template
	var collector *domain.Template

	// The workload follows the collector kind's scope. The operator's resource
	// stands in for the workload and carries the config inline.
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindAgent:
		tmpls = append(tmpls,
			agentServiceaccountTemplate,
			agentClusterroleTemplate,
			agentClusterrolebindingTemplate,
		)
		workload = []*domain.Template{agentServiceTemplate, daemonsetTemplate}
		collector = agentOpenTelemetryCollectorTemplate
	case collectionagent.CollectorKindDeployment:
		tmpls = append(tmpls,
			deploymentServiceaccountTemplate,
			deploymentClusterroleTemplate,
			deploymentClusterrolebindingTemplate,
		)
		workload = []*domain.Template{deploymentServiceTemplate, deploymentTemplate}
		collector = deploymentOpenTelemetryCollectorTemplate
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	switch data.Controller {
	case collectionagent.CollectorControllerDefault:
		tmpls = append(tmpls, workload...)
	case collectionagent.CollectorControllerOpenTelemetryOperator:
		tmpls = append(tmpls, collector)
	default:
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to forge: collector controller %q is not supported, state %q or %q", data.Controller, collectionagent.CollectorControllerDefault, collectionagent.CollectorControllerOpenTelemetryOperator)
	}

	// The kind's directory is a kustomize root of its own, so a casting file
	// holding several collection agents pours a tree per document.
	dir := filepath.Dir(config.Spec.Collector.Kind.ConfigKey())

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

func (c *kubernetesKustomizeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	c.logger.InfoContext(ctx, "Applying kustomize manifests",
		slog.String("release", config.Metadata.Name),
		slog.String("namespace", namespace(config)),
	)

	kustomizeDir := filepath.Join(outputPath, p.Dir(), filepath.Dir(config.Spec.Collector.Kind.ConfigKey()))
	if _, err := os.Stat(filepath.Join(kustomizeDir, "kustomization.yaml")); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "kustomization.yaml does not exist at path: %s, run 'forge' first", kustomizeDir)
	}

	if collectionagent.KubernetesCollectorController.Resolve(config.Metadata.Annotations) == collectionagent.CollectorControllerOpenTelemetryOperator {
		if err := c.kubectl(ctx, "get", "crd", "opentelemetrycollectors.opentelemetry.io", "-o", "name"); err != nil {
			return foundryerrors.Newf(foundryerrors.TypeNotFound, "failed to find the OpenTelemetryCollector CRD: install the OpenTelemetry Operator v0.139.0 (https://signoz.io/docs/opentelemetry-collection-agents/k8s/otel-operator/install/) and rerun")
		}
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

func (c *kubernetesKustomizeCasting) kubectl(ctx context.Context, args ...string) error {
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
