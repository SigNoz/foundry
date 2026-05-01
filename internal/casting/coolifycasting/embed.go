package coolifycasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	coolifyYAMLTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/coolify.yaml.gotmpl", domain.FormatYAML)
)
