package telemetrystoremolding

import (
	"embed"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	ConfigClickhousev2556YAML    *types.Template = types.MustNewTemplateFromFS(templates, "templates/config.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
	FunctionsClickhousev2556YAML *types.Template = types.MustNewTemplateFromFS(templates, "templates/functions.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
	KeeperClickhousev2556YAML    *types.Template = types.MustNewTemplateFromFS(templates, "templates/keeper.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
)

func DefaultSpec() v1alpha1.MoldingSpec {
	return v1alpha1.MoldingSpec{
		Enabled: true,
		Cluster: v1alpha1.TypeCluster{
			Replicas: types.NewIntPtr(0),
			Shards:   types.NewIntPtr(1),
		},
		Version: "25.5.6",
		Image:   "clickhouse/clickhouse-server:25.5.6",
	}
}
