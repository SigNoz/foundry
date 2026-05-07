package systemdcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	signozServiceTemplate                 *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/signoz.service.gotmpl", domain.FormatINI)
	ingesterServiceTemplate               *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/ingester.service.gotmpl", domain.FormatINI)
	telemetryStoreServiceTemplate         *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/clickhouse.telemetrystore.v2556.service.gotmpl", domain.FormatINI)
	telemetryKeeperServiceTemplate        *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/clickhousekeeper.telemetrykeeper.v2556.service.gotmpl", domain.FormatINI)
	metaStoreServiceTemplate              *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/postgres.metastore.service.gotmpl", domain.FormatINI)
	telemetryStoreMigratorServiceTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/migrator.telemetrystore.service.gotmpl", domain.FormatINI)
)
