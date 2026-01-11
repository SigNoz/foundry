package telemetrystoremolding

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	ConfigClickhousev2556YAML    *types.Template = types.MustNewTemplateFromFS(templates, "templates/config.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
	FunctionsClickhousev2556YAML *types.Template = types.MustNewTemplateFromFS(templates, "templates/functions.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
	KeeperClickhousev2556YAML    *types.Template = types.MustNewTemplateFromFS(templates, "templates/keeper.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
)
