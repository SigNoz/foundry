package ecsterraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

// Components are files in the one root module, never child modules: a module
// with a single generated caller is indirection without reuse.
var (
	versionsTF  = domain.MustNewTemplateFromFS(templates, "templates/versions.tf.json.gotmpl", domain.FormatJSON)
	backendTF   = domain.MustNewTemplateFromFS(templates, "templates/backend.tf.json.gotmpl", domain.FormatJSON)
	providersTF = domain.MustNewTemplateFromFS(templates, "templates/providers.tf.json.gotmpl", domain.FormatJSON)
	mainTF      = domain.MustNewTemplateFromFS(templates, "templates/main.tf.json.gotmpl", domain.FormatJSON)
	variablesTF = domain.MustNewTemplateFromFS(templates, "templates/variables.tf.json.gotmpl", domain.FormatJSON)
	outputsTF   = domain.MustNewTemplateFromFS(templates, "templates/outputs.tf.json.gotmpl", domain.FormatJSON)
	tfarsTF     = domain.MustNewTemplateFromFS(templates, "templates/terraform.tfvars.json.gotmpl", domain.FormatJSON)
)

var (
	telemetryKeeperTF = domain.MustNewTemplateFromFS(templates, "templates/telemetrykeeper.tf.json.gotmpl", domain.FormatJSON)
	telemetryStoreTF  = domain.MustNewTemplateFromFS(templates, "templates/telemetrystore.tf.json.gotmpl", domain.FormatJSON)
	migratorTF        = domain.MustNewTemplateFromFS(templates, "templates/telemetrystore_migrator.tf.json.gotmpl", domain.FormatJSON)
	metaStoreTF       = domain.MustNewTemplateFromFS(templates, "templates/metastore.tf.json.gotmpl", domain.FormatJSON)
	signozTF          = domain.MustNewTemplateFromFS(templates, "templates/signoz.tf.json.gotmpl", domain.FormatJSON)
	ingesterTF        = domain.MustNewTemplateFromFS(templates, "templates/ingester.tf.json.gotmpl", domain.FormatJSON)
	mcpTF             = domain.MustNewTemplateFromFS(templates, "templates/mcp.tf.json.gotmpl", domain.FormatJSON)
)
