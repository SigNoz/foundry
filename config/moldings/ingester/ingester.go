package ingester

import (
	"embed"

	"github.com/signoz/foundry/internal/yaml"
)

//go:embed *.yaml
var yamls embed.FS

var (
	ConfigV0129xYAML string = yaml.MustMarshal(yaml.MustFile(yamls, "config.v0129x.yaml"))
)
