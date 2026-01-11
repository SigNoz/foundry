package ingestermolding

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed *.yaml
var yamls embed.FS

var (
	ConfigV0129xYAML string = types.MustMarshalYAML(types.MustNewFileFromFS(yamls, "config.v0129x.yaml"))
)
