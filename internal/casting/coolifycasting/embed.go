package coolifycasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	coolifyYAMLTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/coolify.yaml.gotmpl", types.FormatYAML)
)
