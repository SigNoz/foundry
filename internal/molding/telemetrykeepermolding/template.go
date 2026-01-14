package telemetrykeepermolding

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	KeeperClickhousev2556YAML *types.Template = types.MustNewTemplateFromFS(templates, "templates/keeper.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
)

// Data is the template data for rendering ClickHouse Keeper configs.
type Data struct {
	RaftAddresses   []types.Address // Inter-keeper consensus addresses
	ClientAddresses []types.Address // Client-facing addresses
	ServerCount     int
	ServerID        int // Current server ID for per-node config generation
}
