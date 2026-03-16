package azureacacasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*/*.gotmpl
var templates embed.FS

var (
	// telemetrykeeper.
	telemetrykeeperContainerapp = types.MustNewTemplateFromFS(templates, "templates/telemetrykeeper/containerapp.yaml.gotmpl", types.FormatYAML)

	// telemetrystore.
	telemetrystoreContainerapp = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/containerapp.yaml.gotmpl", types.FormatYAML)

	// metastore.
	metastoreContainerapp = types.MustNewTemplateFromFS(templates, "templates/metastore/containerapp.yaml.gotmpl", types.FormatYAML)

	// signoz.
	signozContainerapp = types.MustNewTemplateFromFS(templates, "templates/signoz/containerapp.yaml.gotmpl", types.FormatYAML)

	// ingester.
	ingesterContainerapp = types.MustNewTemplateFromFS(templates, "templates/ingester/containerapp.yaml.gotmpl", types.FormatYAML)

	// telemetrystore-migrator.
	telemetrystoreMigratorJob = types.MustNewTemplateFromFS(templates, "templates/telemetrystore-migrator/job.yaml.gotmpl", types.FormatYAML)
)
