package telemetrystore

import (
	"embed"

	"github.com/signoz/foundry/internal/yaml"
)

//go:embed *.yaml
var yamls embed.FS

var (
	ConfigClickhousev2556YAML    string = yaml.MustMarshal(yaml.MustFile(yamls, "config.clickhouse.v2556.yaml"))
	FunctionsClickhousev2556YAML string = yaml.MustMarshal(yaml.MustFile(yamls, "functions.clickhouse.v2556.yaml"))
	KeeperClickhousev2556YAML    string = yaml.MustMarshal(yaml.MustFile(yamls, "keeper.clickhouse.v2556.yaml"))
)
