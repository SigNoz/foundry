package ingester

import (
	"embed"

	"github.com/signoz/foundry/config/moldings"
)

//go:embed *.yaml
var yamls embed.FS

var (
	ConfigV0129xYAML string = moldings.MustYAMLMarshal(moldings.MustFile(yamls, "config.v0129x.yaml"))
)
