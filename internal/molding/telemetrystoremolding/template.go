package telemetrystoremolding

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	ConfigClickhousev2556YAML    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/config.clickhouse.v2556.yaml.gotmpl", domain.FormatYAML)
	FunctionsClickhousev2556YAML *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/functions.clickhouse.v2556.yaml.gotmpl", domain.FormatYAML)
)

// ConfigClickhousev2556ListMerge declares how a user override merges with the
// output of ConfigClickhousev2556YAML, per list path. Cluster topology
// (remote_servers, zookeeper nodes) is Foundry-owned and derived from
// addresses, so an override replaces those lists wholesale. ClickHouse has no
// union-style lists, so every path is atomic — also the default; the entries
// state intent and keep these object-lists off Set/Ordered. It lives beside the
// template because it describes that template's rendered shape.
var ConfigClickhousev2556ListMerge = map[string]domain.ListMerge{
	"remote_servers.*.shard": domain.ListMergeReplace,
	"zookeeper.node":         domain.ListMergeReplace,
}

// Data is the template data for rendering ClickHouse telemetry store configs.
type Data struct {
	StoreAddresses  []domain.Address
	KeeperAddresses []domain.Address
	ShardCount      int
	ReplicaCount    int
	ShardID         int // 0-indexed, used to render per-node macros.shard
	ReplicaID       int // 0-indexed, used to render per-node macros.replica
}
