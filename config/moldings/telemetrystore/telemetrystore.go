package telemetrystore

import (
	"embed"

	"github.com/signoz/foundry/config/moldings"
)

//go:embed *.yaml
var yamls embed.FS

var (
	ConfigClickhousev2556YAML    string = moldings.MustYAMLMarshal(moldings.MustFile(yamls, "config.clickhouse.v2556.yaml"))
	FunctionsClickhousev2556YAML string = moldings.MustYAMLMarshal(moldings.MustFile(yamls, "functions.clickhouse.v2556.yaml"))
	KeeperClickhousev2556YAML    string = moldings.MustYAMLMarshal(moldings.MustFile(yamls, "keeper.clickhouse.v2556.yaml"))
)
