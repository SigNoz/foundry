package telemetrystoremolding

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	ConfigClickhousev25125YAML    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/config.clickhouse.v25125.yaml.gotmpl", domain.FormatYAML)
	FunctionsClickhousev25125YAML *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/functions.clickhouse.v25125.yaml.gotmpl", domain.FormatYAML)
	UsersClickhousev25125YAML     *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/users.clickhouse.v25125.yaml.gotmpl", domain.FormatYAML)
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
