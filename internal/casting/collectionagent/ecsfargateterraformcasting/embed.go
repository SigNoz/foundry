package ecsfargateterraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/sidecar/*.gotmpl
var templates embed.FS

var (
	versionsTF = domain.MustNewTemplateFromFS(templates, "templates/sidecar/versions.tf.json.gotmpl", domain.FormatJSON)
	mainTF     = domain.MustNewTemplateFromFS(templates, "templates/sidecar/main.tf.json.gotmpl", domain.FormatJSON)
	outputsTF  = domain.MustNewTemplateFromFS(templates, "templates/sidecar/outputs.tf.json.gotmpl", domain.FormatJSON)

	sidecarYAMLTemplate = domain.MustNewTemplateFromFS(templates, "templates/sidecar/sidecar.yaml.gotmpl", domain.FormatYAML)
)
