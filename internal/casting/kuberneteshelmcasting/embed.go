package kuberneteshelmcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/values.yaml.gotmpl
var templates embed.FS

var valuesYAMLTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/values.yaml.gotmpl", types.FormatYAML)
