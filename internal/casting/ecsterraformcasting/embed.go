package ecsterraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl templates/module/*.gotmpl
var templates embed.FS

// Root Terraform templates.
var (
	mainTF      = domain.MustNewTemplateFromFS(templates, "templates/main.tf.json.gotmpl", domain.FormatJSON)
	variablesTF = domain.MustNewTemplateFromFS(templates, "templates/variables.tf.json.gotmpl", domain.FormatJSON)
	tfarsTF     = domain.MustNewTemplateFromFS(templates, "templates/terraform.tfvars.json.gotmpl", domain.FormatJSON)
)

// Module Terraform templates.
var (
	moduleMainTF      = domain.MustNewTemplateFromFS(templates, "templates/module/main.tf.json.gotmpl", domain.FormatJSON)
	moduleVariablesTF = domain.MustNewTemplateFromFS(templates, "templates/module/variables.tf.json.gotmpl", domain.FormatJSON)
	moduleOutputsTF   = domain.MustNewTemplateFromFS(templates, "templates/module/outputs.tf.json.gotmpl", domain.FormatJSON)

	moduleTelemetryKeeperTF = domain.MustNewTemplateFromFS(templates, "templates/module/telemetrykeeper.tf.json.gotmpl", domain.FormatJSON)
	moduleTelemetryStoreTF  = domain.MustNewTemplateFromFS(templates, "templates/module/telemetrystore.tf.json.gotmpl", domain.FormatJSON)
	moduleMigratorTF        = domain.MustNewTemplateFromFS(templates, "templates/module/telemetrystore_migrator.tf.json.gotmpl", domain.FormatJSON)
	moduleMetaStoreTF       = domain.MustNewTemplateFromFS(templates, "templates/module/metastore.tf.json.gotmpl", domain.FormatJSON)
	moduleSignozTF          = domain.MustNewTemplateFromFS(templates, "templates/module/signoz.tf.json.gotmpl", domain.FormatJSON)
	moduleIngesterTF        = domain.MustNewTemplateFromFS(templates, "templates/module/ingester.tf.json.gotmpl", domain.FormatJSON)
)
