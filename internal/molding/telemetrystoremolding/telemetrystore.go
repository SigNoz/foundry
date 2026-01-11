package telemetrystoremolding

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed *.yaml
var yamls embed.FS

var (
	ConfigClickhousev2556YAML    string = types.MustMarshalYAML(types.MustNewFileFromFS(yamls, "config.clickhouse.v2556.yaml"))
	FunctionsClickhousev2556YAML string = types.MustMarshalYAML(types.MustNewFileFromFS(yamls, "functions.clickhouse.v2556.yaml"))
	KeeperClickhousev2556YAML    string = types.MustMarshalYAML(types.MustNewFileFromFS(yamls, "keeper.clickhouse.v2556.yaml"))
)
