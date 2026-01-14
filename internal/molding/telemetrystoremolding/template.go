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
)

// Data is the template data for rendering ClickHouse telemetry store configs.
type Data struct {
	StoreAddresses  []types.Address
	KeeperAddresses []types.Address
	ShardCount      int
	ReplicaCount    int
}
