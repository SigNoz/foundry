package linuxcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	signozServiceTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/signoz.service.gotmpl", types.FormatINI)
	ingesterServiceTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/ingester.service.gotmpl", types.FormatINI)

)
