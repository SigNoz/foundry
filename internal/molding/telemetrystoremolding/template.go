package telemetrystoremolding

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	configClickhousev2556YAML  *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/config.clickhouse.v2556.yaml.gotmpl", domain.FormatYAML)
	configClickhousev25125YAML *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/config.clickhouse.v25125.yaml.gotmpl", domain.FormatYAML)

	functionsClickhousev2556YAML  *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/functions.clickhouse.v2556.yaml.gotmpl", domain.FormatYAML)
	functionsClickhousev25125YAML *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/functions.clickhouse.v25125.yaml.gotmpl", domain.FormatYAML)

	// clickhouseConfigTemplates selects the per-node ClickHouse server config by
	// the configured store version, falling back to the latest when unknown.
	clickhouseConfigTemplates = domain.NewVersionedTemplates("25.12.5", map[string]*domain.Template{
		"25.5.6":  configClickhousev2556YAML,
		"25.12.5": configClickhousev25125YAML,
	})

	// clickhouseFunctionsTemplates selects the ClickHouse UDF functions config by
	// the configured store version, falling back to the latest when unknown.
	clickhouseFunctionsTemplates = domain.NewVersionedTemplates("25.12.5", map[string]*domain.Template{
		"25.5.6":  functionsClickhousev2556YAML,
		"25.12.5": functionsClickhousev25125YAML,
	})
)

// Data is the template data for rendering ClickHouse telemetry store configs.
type Data struct {
	StoreAddresses  []domain.Address
	KeeperAddresses []domain.Address
	ShardCount      int
	ReplicaCount    int
	ShardID         int // 0-indexed, used to render per-node macros.shard
	ReplicaID       int // 0-indexed, used to render per-node macros.replica
}
