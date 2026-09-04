package ecsec2terraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	versionsTFTemplate  *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/versions.tf.json.gotmpl", domain.FormatJSON)
	backendTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/backend.tf.json.gotmpl", domain.FormatJSON)
	providersTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/providers.tf.json.gotmpl", domain.FormatJSON)
	mainTFTemplate      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/main.tf.json.gotmpl", domain.FormatJSON)
	variablesTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/variables.tf.json.gotmpl", domain.FormatJSON)
	outputsTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/outputs.tf.json.gotmpl", domain.FormatJSON)
	cloudInitTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/cloudinit.yaml.gotmpl", domain.FormatText)
)
