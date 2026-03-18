package ecstaskdefcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*/*.gotmpl templates/*/*/*.gotmpl
var templates embed.FS

var (
	telemetryKeeperTaskDefinition = types.MustNewTemplateFromFS(templates, "templates/telemetrykeeper/clickhousekeeper/task-definition.json.gotmpl", types.FormatText)
	telemetryStoreTaskDefinition  = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse/task-definition.json.gotmpl", types.FormatText)
	metaStoreTaskDefinition       = types.MustNewTemplateFromFS(templates, "templates/metastore/postgres/task-definition.json.gotmpl", types.FormatText)
	migratorTaskDefinition        = types.MustNewTemplateFromFS(templates, "templates/telemetrystore-migrator/task-definition.json.gotmpl", types.FormatText)
	signozTaskDefinition          = types.MustNewTemplateFromFS(templates, "templates/signoz/task-definition.json.gotmpl", types.FormatText)
	ingesterTaskDefinition        = types.MustNewTemplateFromFS(templates, "templates/ingester/task-definition.json.gotmpl", types.FormatText)
)
