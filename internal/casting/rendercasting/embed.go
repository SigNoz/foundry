package rendercasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	renderYAMLTemplate                *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/render.yaml.gotmpl", domain.FormatYAML)
	telemetryKeeperDockerfileTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhousekeeper.telemetrykeeper.v2556.gotmpl", domain.FormatText)
	telemetryStoreDockerfileTemplate  *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhouse.telemetrystore.v2556.gotmpl", domain.FormatText)
	ingesterDockerfileTemplate        *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.ingester.gotmpl", domain.FormatText)
)
