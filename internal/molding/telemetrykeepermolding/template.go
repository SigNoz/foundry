package telemetrykeepermolding

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	KeeperClickhousev2556YAML    *types.Template = types.MustNewTemplateFromFS(templates, "templates/keeper.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
)

type Data struct {
	TelemetryKeeperClickhouseCluster RaftConfig
	ServerID int
}

type RaftConfig struct{
	Servers []Server `json:"server" yaml:"server"`
}

type Server struct {
    Host string `json:"hostname" yaml:"hostname"`
    Port string    `json:"port" yaml:"port"`
	ID int `json:"id" yaml:"id"`
}