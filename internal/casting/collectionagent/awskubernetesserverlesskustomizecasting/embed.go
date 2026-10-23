package awskubernetesserverlesskustomizecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl templates/deployment/*.gotmpl templates/sidecar/*.gotmpl
var templates embed.FS

var (
	kustomizationTemplate = domain.MustNewTemplateFromFS(templates, "templates/kustomization.yaml.gotmpl", domain.FormatYAML)
	namespaceTemplate     = domain.MustNewTemplateFromFS(templates, "templates/namespace.yaml.gotmpl", domain.FormatYAML)

	deploymentServiceaccountTemplate     = domain.MustNewTemplateFromFS(templates, "templates/deployment/serviceaccount.yaml.gotmpl", domain.FormatYAML)
	deploymentClusterroleTemplate        = domain.MustNewTemplateFromFS(templates, "templates/deployment/clusterrole.yaml.gotmpl", domain.FormatYAML)
	deploymentClusterrolebindingTemplate = domain.MustNewTemplateFromFS(templates, "templates/deployment/clusterrolebinding.yaml.gotmpl", domain.FormatYAML)
	deploymentServiceTemplate            = domain.MustNewTemplateFromFS(templates, "templates/deployment/service.yaml.gotmpl", domain.FormatYAML)
	deploymentTemplate                   = domain.MustNewTemplateFromFS(templates, "templates/deployment/workload.yaml.gotmpl", domain.FormatYAML)
	deploymentYAMLTemplate               = domain.MustNewTemplateFromFS(templates, "templates/deployment/collector.yaml.gotmpl", domain.FormatYAML)

	deploymentOpenTelemetryCollectorTemplate = domain.MustNewTemplateFromFS(templates, "templates/deployment/opentelemetrycollector.yaml.gotmpl", domain.FormatYAML)

	sidecarKustomizationTemplate          = domain.MustNewTemplateFromFS(templates, "templates/sidecar/kustomization.yaml.gotmpl", domain.FormatYAML)
	sidecarRoleTemplate                   = domain.MustNewTemplateFromFS(templates, "templates/sidecar/role.yaml.gotmpl", domain.FormatYAML)
	sidecarRolebindingTemplate            = domain.MustNewTemplateFromFS(templates, "templates/sidecar/rolebinding.yaml.gotmpl", domain.FormatYAML)
	sidecarPodtemplateTemplate            = domain.MustNewTemplateFromFS(templates, "templates/sidecar/podtemplate.yaml.gotmpl", domain.FormatYAML)
	sidecarPatchTemplate                  = domain.MustNewTemplateFromFS(templates, "templates/sidecar/patch.yaml.gotmpl", domain.FormatYAML)
	sidecarPatchCronjobTemplate           = domain.MustNewTemplateFromFS(templates, "templates/sidecar/patch-cronjob.yaml.gotmpl", domain.FormatYAML)
	sidecarOpenTelemetryCollectorTemplate = domain.MustNewTemplateFromFS(templates, "templates/sidecar/opentelemetrycollector.yaml.gotmpl", domain.FormatYAML)
	sidecarYAMLTemplate                   = domain.MustNewTemplateFromFS(templates, "templates/sidecar/collector.yaml.gotmpl", domain.FormatYAML)
)
