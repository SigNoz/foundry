package railwaytemplatecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	telemetryKeeperDockerfileTemplate        *types.Template = types.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhousekeeper.telemetrykeeper.v2556.gotmpl", types.FormatText)
	telemetryStoreDockerfileTemplate         *types.Template = types.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhouse.telemetrystore.v2556.gotmpl", types.FormatText)
	ingesterDockerfileTemplate               *types.Template = types.MustNewTemplateFromFS(templates, "templates/Dockerfile.ingester.gotmpl", types.FormatText)
	signozDockerfileTemplate                 *types.Template = types.MustNewTemplateFromFS(templates, "templates/Dockerfile.signoz.gotmpl", types.FormatText)
	telemetryStoreMigratorDockerfileTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/Dockerfile.telemetrystore-migrator.gotmpl", types.FormatText)

	railwayTelemetryKeeperTemplate        *types.Template = types.MustNewTemplateFromFS(templates, "templates/railway.telemetrykeeper.json.gotmpl", types.FormatText)
	railwayTelemetryStoreTemplate         *types.Template = types.MustNewTemplateFromFS(templates, "templates/railway.telemetrystore.json.gotmpl", types.FormatText)
	railwayIngesterTemplate               *types.Template = types.MustNewTemplateFromFS(templates, "templates/railway.ingester.json.gotmpl", types.FormatText)
	railwaySignozTemplate                 *types.Template = types.MustNewTemplateFromFS(templates, "templates/railway.signoz.json.gotmpl", types.FormatText)
	railwayTelemetryStoreMigratorTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/railway.telemetrystore-migrator.json.gotmpl", types.FormatText)

	// molding overrides.
	telemetryKeeperOverrideTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/telemetrykeeper.yaml.gotmpl", types.FormatYAML)
	telemetryStoreOverrideTemplate  *types.Template = types.MustNewTemplateFromFS(templates, "templates/telemetrystore.yaml.gotmpl", types.FormatYAML)
)
