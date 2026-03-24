package ecsterraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl templates/module/*.gotmpl
var templates embed.FS

// Root Terraform templates.
var (
	mainTF      = types.MustNewTemplateFromFS(templates, "templates/main.tf.json.gotmpl", types.FormatJSON)
	variablesTF = types.MustNewTemplateFromFS(templates, "templates/variables.tf.json.gotmpl", types.FormatJSON)
	tfarsTF     = types.MustNewTemplateFromFS(templates, "templates/terraform.tfvars.json.gotmpl", types.FormatJSON)
)

// Module Terraform templates.
var (
	moduleMainTF      = types.MustNewTemplateFromFS(templates, "templates/module/main.tf.json.gotmpl", types.FormatJSON)
	moduleVariablesTF = types.MustNewTemplateFromFS(templates, "templates/module/variables.tf.json.gotmpl", types.FormatJSON)
	moduleOutputsTF   = types.MustNewTemplateFromFS(templates, "templates/module/outputs.tf.json.gotmpl", types.FormatJSON)

	moduleTelemetryKeeperTF = types.MustNewTemplateFromFS(templates, "templates/module/telemetrykeeper.tf.json.gotmpl", types.FormatJSON)
	moduleTelemetryStoreTF  = types.MustNewTemplateFromFS(templates, "templates/module/telemetrystore.tf.json.gotmpl", types.FormatJSON)
	moduleMigratorTF        = types.MustNewTemplateFromFS(templates, "templates/module/telemetrystore_migrator.tf.json.gotmpl", types.FormatJSON)
	moduleMetaStoreTF       = types.MustNewTemplateFromFS(templates, "templates/module/metastore.tf.json.gotmpl", types.FormatJSON)
	moduleSignozTF          = types.MustNewTemplateFromFS(templates, "templates/module/signoz.tf.json.gotmpl", types.FormatJSON)
	moduleIngesterTF        = types.MustNewTemplateFromFS(templates, "templates/module/ingester.tf.json.gotmpl", types.FormatJSON)
)
