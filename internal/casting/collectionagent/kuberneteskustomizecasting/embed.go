package kuberneteskustomizecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl templates/agent/*.gotmpl templates/deployment/*.gotmpl
var templates embed.FS

var (
	kustomizationTemplate = domain.MustNewTemplateFromFS(templates, "templates/kustomization.yaml.gotmpl", domain.FormatYAML)
	namespaceTemplate     = domain.MustNewTemplateFromFS(templates, "templates/namespace.yaml.gotmpl", domain.FormatYAML)

	agentServiceaccountTemplate     = domain.MustNewTemplateFromFS(templates, "templates/agent/serviceaccount.yaml.gotmpl", domain.FormatYAML)
	agentClusterroleTemplate        = domain.MustNewTemplateFromFS(templates, "templates/agent/clusterrole.yaml.gotmpl", domain.FormatYAML)
	agentClusterrolebindingTemplate = domain.MustNewTemplateFromFS(templates, "templates/agent/clusterrolebinding.yaml.gotmpl", domain.FormatYAML)
	agentServiceTemplate            = domain.MustNewTemplateFromFS(templates, "templates/agent/service.yaml.gotmpl", domain.FormatYAML)
	daemonsetTemplate               = domain.MustNewTemplateFromFS(templates, "templates/agent/workload.yaml.gotmpl", domain.FormatYAML)
	agentYAMLTemplate               = domain.MustNewTemplateFromFS(templates, "templates/agent/collector.yaml.gotmpl", domain.FormatYAML)

	deploymentServiceaccountTemplate     = domain.MustNewTemplateFromFS(templates, "templates/deployment/serviceaccount.yaml.gotmpl", domain.FormatYAML)
	deploymentClusterroleTemplate        = domain.MustNewTemplateFromFS(templates, "templates/deployment/clusterrole.yaml.gotmpl", domain.FormatYAML)
	deploymentClusterrolebindingTemplate = domain.MustNewTemplateFromFS(templates, "templates/deployment/clusterrolebinding.yaml.gotmpl", domain.FormatYAML)
	deploymentServiceTemplate            = domain.MustNewTemplateFromFS(templates, "templates/deployment/service.yaml.gotmpl", domain.FormatYAML)
	deploymentTemplate                   = domain.MustNewTemplateFromFS(templates, "templates/deployment/workload.yaml.gotmpl", domain.FormatYAML)
	deploymentYAMLTemplate               = domain.MustNewTemplateFromFS(templates, "templates/deployment/collector.yaml.gotmpl", domain.FormatYAML)
)
