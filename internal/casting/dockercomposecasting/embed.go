package dockercomposecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	composeYAMLTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/compose.yaml.gotmpl", types.FormatYAML)
)
