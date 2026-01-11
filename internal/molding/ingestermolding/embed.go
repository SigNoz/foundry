package ingestermolding

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	ConfigV0129xTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/config.v0129x.yaml.gotmpl")
)
