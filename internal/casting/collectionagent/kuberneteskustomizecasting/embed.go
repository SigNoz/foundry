package kuberneteskustomizecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

// The pour is one kustomize root: kustomization.yaml is the entry point, and
// the collector config enters through a configMapGenerator so a config change
// re-hashes the ConfigMap name and rolls the workload.
var (
	kustomizationTemplate      = domain.MustNewTemplateFromFS(templates, "templates/kustomization.yaml.gotmpl", domain.FormatYAML)
	namespaceTemplate          = domain.MustNewTemplateFromFS(templates, "templates/namespace.yaml.gotmpl", domain.FormatYAML)
	serviceaccountTemplate     = domain.MustNewTemplateFromFS(templates, "templates/serviceaccount.yaml.gotmpl", domain.FormatYAML)
	clusterroleTemplate        = domain.MustNewTemplateFromFS(templates, "templates/clusterrole.yaml.gotmpl", domain.FormatYAML)
	clusterrolebindingTemplate = domain.MustNewTemplateFromFS(templates, "templates/clusterrolebinding.yaml.gotmpl", domain.FormatYAML)
	daemonsetTemplate          = domain.MustNewTemplateFromFS(templates, "templates/daemonset.yaml.gotmpl", domain.FormatYAML)
	deploymentTemplate         = domain.MustNewTemplateFromFS(templates, "templates/deployment.yaml.gotmpl", domain.FormatYAML)
	serviceTemplate            = domain.MustNewTemplateFromFS(templates, "templates/service.yaml.gotmpl", domain.FormatYAML)

	// Enricher config templates, one per collector kind, each named by the
	// config file it renders. The collector. prefix keeps them apart from
	// the deployment manifest sharing this directory.
	agentYAMLTemplate      = domain.MustNewTemplateFromFS(templates, "templates/collector.agent.yaml.gotmpl", domain.FormatYAML)
	deploymentYAMLTemplate = domain.MustNewTemplateFromFS(templates, "templates/collector.deployment.yaml.gotmpl", domain.FormatYAML)
)
