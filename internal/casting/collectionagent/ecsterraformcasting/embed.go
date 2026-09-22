package ecsterraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	versionsTF  = domain.MustNewTemplateFromFS(templates, "templates/versions.tf.json.gotmpl", domain.FormatJSON)
	providersTF = domain.MustNewTemplateFromFS(templates, "templates/providers.tf.json.gotmpl", domain.FormatJSON)
	backendTF   = domain.MustNewTemplateFromFS(templates, "templates/backend.tf.json.gotmpl", domain.FormatJSON)
	variablesTF = domain.MustNewTemplateFromFS(templates, "templates/variables.tf.json.gotmpl", domain.FormatJSON)
	tfvarsTF    = domain.MustNewTemplateFromFS(templates, "templates/terraform.tfvars.json.gotmpl", domain.FormatJSON)
	mainTF      = domain.MustNewTemplateFromFS(templates, "templates/main.tf.json.gotmpl", domain.FormatJSON)
	collectorTF = domain.MustNewTemplateFromFS(templates, "templates/collector.tf.json.gotmpl", domain.FormatJSON)

	agentYAMLTemplate = domain.MustNewTemplateFromFS(templates, "templates/agent.yaml.gotmpl", domain.FormatYAML)
)
