package kuberneteskustomizecasting

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
	tmpls := []*domain.Template{kustomizationTemplate, namespaceTemplate}

	// The workload follows the collector kind's scope: the agent runs on
	// every node, the deployment runs replicated behind the service.
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindAgent:
		tmpls = append(tmpls,
			agentServiceaccountTemplate,
			agentClusterroleTemplate,
			agentClusterrolebindingTemplate,
			agentServiceTemplate,
			daemonsetTemplate,
		)
	case collectionagent.CollectorKindDeployment:
		tmpls = append(tmpls,
			deploymentServiceaccountTemplate,
			deploymentClusterroleTemplate,
			deploymentClusterrolebindingTemplate,
			deploymentServiceTemplate,
			deploymentTemplate,
		)
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	// The kind's directory is a kustomize root of its own, so a casting file
	// holding several collection agents pours a tree per document.
	dir := filepath.Dir(config.Spec.Collector.Kind.ConfigKey())

	for _, tmpl := range tmpls {
		buf := bytes.NewBuffer(nil)
		if err := tmpl.Execute(buf, config); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", tmpl.Name())
		}

		p.AddYAML(buf.Bytes(), dir, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
	}

	// The collector config, inside the kustomize root so the configMapGenerator
	// reaches it by relative path.
	for path, content := range config.Spec.Collector.Spec.Config.Data {
		p.AddYAML([]byte(content), path)
	}

	return nil
}

func (c *kubernetesKustomizeCasting) Cast(ctx context.Context, config collectionagent.Casting, outputPath string, p *pourer.Pourer) error {
	kustomizeDir := filepath.Join(outputPath, p.Dir(), filepath.Dir(config.Spec.Collector.Kind.ConfigKey()))

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
