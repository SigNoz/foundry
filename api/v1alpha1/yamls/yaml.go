package yamls

import (
	"embed"

	"go.yaml.in/yaml/v3"
)

//go:embed *.yaml
var yamls embed.FS

var (
	ConfigIngesterV0129xYAML     string = MustYAMLMarshal(MustFile("config.ingester.v0129x.yaml"))
	ConfigClickhousev2556YAML    string = MustYAMLMarshal(MustFile("config.clickhouse.v2556.yaml"))
	FunctionsClickhousev2556YAML string = MustYAMLMarshal(MustFile("functions.clickhouse.v2556.yaml"))
	KeeperClickhousev2556YAML    string = MustYAMLMarshal(MustFile("keeper.clickhouse.v2556.yaml"))
)

func MustFile(name string) string {
	data, err := yamls.ReadFile(name)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func MustYAMLMarshal(v any) string {
	yaml, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(yaml)
}
