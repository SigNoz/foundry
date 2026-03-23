package ecstaskdefcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/root/*.gotmpl templates/module/*.gotmpl
var templates embed.FS

// Root Terraform templates
var (
	rootMainTF      = types.MustNewTemplateFromFS(templates, "templates/root/main.tf.json.gotmpl", types.FormatText)
	rootVariablesTF = types.MustNewTemplateFromFS(templates, "templates/root/variables.tf.json.gotmpl", types.FormatText)
	rootTfvarsTF    = types.MustNewTemplateFromFS(templates, "templates/root/terraform.tfvars.json.gotmpl", types.FormatJSON)
)

// Module Terraform templates
var (
	moduleMainTF      = types.MustNewTemplateFromFS(templates, "templates/module/main.tf.json.gotmpl", types.FormatText)
	moduleVariablesTF = types.MustNewTemplateFromFS(templates, "templates/module/variables.tf.json.gotmpl", types.FormatText)
	moduleOutputsTF   = types.MustNewTemplateFromFS(templates, "templates/module/outputs.tf.json.gotmpl", types.FormatText)

	moduleTelemetryKeeperTF = types.MustNewTemplateFromFS(templates, "templates/module/telemetrykeeper.tf.json.gotmpl", types.FormatText)
	moduleTelemetryStoreTF  = types.MustNewTemplateFromFS(templates, "templates/module/telemetrystore.tf.json.gotmpl", types.FormatText)
	moduleMigratorTF        = types.MustNewTemplateFromFS(templates, "templates/module/telemetrystore_migrator.tf.json.gotmpl", types.FormatText)
	moduleMetaStoreTF       = types.MustNewTemplateFromFS(templates, "templates/module/metastore.tf.json.gotmpl", types.FormatText)
	moduleSignozTF          = types.MustNewTemplateFromFS(templates, "templates/module/signoz.tf.json.gotmpl", types.FormatText)
	moduleIngesterTF        = types.MustNewTemplateFromFS(templates, "templates/module/ingester.tf.json.gotmpl", types.FormatText)
)
