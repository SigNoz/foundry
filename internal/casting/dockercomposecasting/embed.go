package dockercomposecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	composeYAMLTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/compose.yaml.gotmpl", domain.FormatYAML)
)
