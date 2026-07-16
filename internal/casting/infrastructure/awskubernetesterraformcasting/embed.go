package awskubernetesterraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	providersTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/providers.tf.json.gotmpl", domain.FormatJSON)
	mainTFTemplate      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/main.tf.json.gotmpl", domain.FormatJSON)
	variablesTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/variables.tf.json.gotmpl", domain.FormatJSON)
	outputsTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/outputs.tf.json.gotmpl", domain.FormatJSON)
)
