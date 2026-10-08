package awsecsterraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	versionsTF  = domain.MustNewTemplateFromFS(templates, "templates/versions.tf.json.gotmpl", domain.FormatJSON)
	backendTF   = domain.MustNewTemplateFromFS(templates, "templates/backend.tf.json.gotmpl", domain.FormatJSON)
	providersTF = domain.MustNewTemplateFromFS(templates, "templates/providers.tf.json.gotmpl", domain.FormatJSON)
	mainTF      = domain.MustNewTemplateFromFS(templates, "templates/main.tf.json.gotmpl", domain.FormatJSON)
	variablesTF = domain.MustNewTemplateFromFS(templates, "templates/variables.tf.json.gotmpl", domain.FormatJSON)
	outputsTF   = domain.MustNewTemplateFromFS(templates, "templates/outputs.tf.json.gotmpl", domain.FormatJSON)

	cloudInitYAML = domain.MustNewTemplateFromFS(templates, "templates/cloudinit.yaml.gotmpl", domain.FormatText)
)
