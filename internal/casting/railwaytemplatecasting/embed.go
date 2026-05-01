package railwaytemplatecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	telemetryKeeperDockerfileTemplate        *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhousekeeper.telemetrykeeper.v2556.gotmpl", domain.FormatText)
	telemetryStoreDockerfileTemplate         *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhouse.telemetrystore.v2556.gotmpl", domain.FormatText)
	ingesterDockerfileTemplate               *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.ingester.gotmpl", domain.FormatText)
	signozDockerfileTemplate                 *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.signoz.gotmpl", domain.FormatText)
	telemetryStoreMigratorDockerfileTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.telemetrystore-migrator.gotmpl", domain.FormatText)

	railwayTelemetryKeeperTemplate        *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/railway.telemetrykeeper.json.gotmpl", domain.FormatText)
	railwayTelemetryStoreTemplate         *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/railway.telemetrystore.json.gotmpl", domain.FormatText)
	railwayIngesterTemplate               *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/railway.ingester.json.gotmpl", domain.FormatText)
	railwaySignozTemplate                 *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/railway.signoz.json.gotmpl", domain.FormatText)
	railwayTelemetryStoreMigratorTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/railway.telemetrystore-migrator.json.gotmpl", domain.FormatText)

	// molding overrides.
	telemetryKeeperOverrideTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/telemetrykeeper.yaml.gotmpl", domain.FormatYAML)
	telemetryStoreOverrideTemplate  *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/telemetrystore.yaml.gotmpl", domain.FormatYAML)
)
