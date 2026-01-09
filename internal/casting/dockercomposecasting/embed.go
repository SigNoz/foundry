package dockercomposecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/template"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	ComposeYAMLTemplate *template.Template = template.MustNew(templates, "templates/compose.yaml.gotmpl")
)
