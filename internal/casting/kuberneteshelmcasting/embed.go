package kuberneteshelmcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/values.yaml.gotmpl templates/values.mcp.yaml.gotmpl
var templates embed.FS

var (
	valuesYAMLTemplate    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/values.yaml.gotmpl", domain.FormatYAML)
	mcpValuesYAMLTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/values.mcp.yaml.gotmpl", domain.FormatYAML)
)
