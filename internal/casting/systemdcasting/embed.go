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
	metaStoreServiceTemplate              *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/postgres.metastore.service.gotmpl", domain.FormatINI)
	telemetryStoreMigratorServiceTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/migrator.telemetrystore.service.gotmpl", domain.FormatINI)
	mcpServiceTemplate                    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/mcp.service.gotmpl", domain.FormatINI)

	// Version-stamped ClickHouse units, dispatched by the configured store/keeper version.
	telemetryStoreServicev2556     *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/clickhouse.telemetrystore.v2556.service.gotmpl", domain.FormatINI)
	telemetryStoreServicev25125    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/clickhouse.telemetrystore.v25125.service.gotmpl", domain.FormatINI)
	telemetryStoreServiceTemplates                  = domain.MustNewVersionedTemplates("25.12.5", map[string]*domain.Template{
		"25.5.6":  telemetryStoreServicev2556,
		"25.12.5": telemetryStoreServicev25125,
	})

	telemetryKeeperServicev2556     *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/clickhousekeeper.telemetrykeeper.v2556.service.gotmpl", domain.FormatINI)
	telemetryKeeperServicev25125    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/clickhousekeeper.telemetrykeeper.v25125.service.gotmpl", domain.FormatINI)
	telemetryKeeperServiceTemplates                  = domain.MustNewVersionedTemplates("25.12.5", map[string]*domain.Template{
		"25.5.6":  telemetryKeeperServicev2556,
		"25.12.5": telemetryKeeperServicev25125,
	})
)
