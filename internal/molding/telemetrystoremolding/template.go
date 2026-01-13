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

type Data struct {
	TelemetryStoreClickHouseCluster ClusterConfig
}

type ClusterConfig struct {
    Shards []ShardConfig `json:"shard" yaml:"shard"`
}

type ShardConfig struct {
    Replicas []ReplicaConfig `json:"replica" yaml:"replica"`
}

type ReplicaConfig struct {
    Host string `json:"host" yaml:"host"`
    Port string    `json:"port" yaml:"port"`
}