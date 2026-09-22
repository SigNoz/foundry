package ecsec2terraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/agent/*.gotmpl
var templates embed.FS

var (
	versionsTF  = domain.MustNewTemplateFromFS(templates, "templates/agent/versions.tf.json.gotmpl", domain.FormatJSON)
	providersTF = domain.MustNewTemplateFromFS(templates, "templates/agent/providers.tf.json.gotmpl", domain.FormatJSON)
	backendTF   = domain.MustNewTemplateFromFS(templates, "templates/agent/backend.tf.json.gotmpl", domain.FormatJSON)
	variablesTF = domain.MustNewTemplateFromFS(templates, "templates/agent/variables.tf.json.gotmpl", domain.FormatJSON)
	tfvarsTF    = domain.MustNewTemplateFromFS(templates, "templates/agent/terraform.tfvars.json.gotmpl", domain.FormatJSON)
	mainTF      = domain.MustNewTemplateFromFS(templates, "templates/agent/main.tf.json.gotmpl", domain.FormatJSON)
	collectorTF = domain.MustNewTemplateFromFS(templates, "templates/agent/collector.tf.json.gotmpl", domain.FormatJSON)

	agentYAMLTemplate = domain.MustNewTemplateFromFS(templates, "templates/agent/agent.yaml.gotmpl", domain.FormatYAML)
)
