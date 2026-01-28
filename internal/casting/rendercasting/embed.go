package rendercasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	renderYAMLTemplate                *types.Template = types.MustNewTemplateFromFS(templates, "templates/render.yaml.gotmpl", types.FormatYAML)
	telemetryKeeperDockerfileTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhousekeeper.telemetrykeeper.v2556.gotmpl", types.FormatText)
	telemetryStoreDockerfileTemplate  *types.Template = types.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhouse.telemetrystore.v2556.gotmpl", types.FormatText)
	ingesterDockerfileTemplate        *types.Template = types.MustNewTemplateFromFS(templates, "templates/Dockerfile.ingester.gotmpl", types.FormatText)
)
