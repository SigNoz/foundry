package kuberneteskustomizecasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

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
	type item struct {
		template *domain.Template
		path     string
	}

	items := []item{
		{kustomizationTemplate, "kustomization.yaml"},
		{namespaceTemplate, "namespace.yaml"},
		{serviceaccountTemplate, "serviceaccount.yaml"},
		{clusterroleTemplate, "clusterrole.yaml"},
		{clusterrolebindingTemplate, "clusterrolebinding.yaml"},
		{serviceTemplate, "service.yaml"},
	}

	// The workload follows the collector kind's scope: the agent runs on
	// every node, the deployment runs replicated behind the service.
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindAgent:
		items = append(items, item{daemonsetTemplate, "daemonset.yaml"})
	case collectionagent.CollectorKindDeployment:
		items = append(items, item{deploymentTemplate, "deployment.yaml"})
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	for _, item := range items {
		buf := bytes.NewBuffer(nil)
		if err := item.template.Execute(buf, config); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", item.path)
		}

		p.AddYAML(buf.Bytes(), item.path)
	}

	// The collector config, inside the kustomize root so the configMapGenerator
	// reaches it by relative path.
	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *kubernetesKustomizeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	kustomizeDir := filepath.Join(outputPath, p.Dir())

	if _, err := os.Stat(filepath.Join(kustomizeDir, "kustomization.yaml")); os.IsNotExist(err) {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "kustomization.yaml does not exist at path: %s, run 'forge' first", kustomizeDir)
	}

	c.logger.InfoContext(ctx, "Applying kustomize manifests", slog.String("path", kustomizeDir))

	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-k", kustomizeDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "kubectl apply -k failed")
	}

	return nil
}
