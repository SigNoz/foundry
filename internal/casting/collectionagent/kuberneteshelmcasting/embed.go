package kuberneteshelmcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl templates/agent/*.gotmpl templates/deployment/*.gotmpl
var templates embed.FS

var (
	valuesYAMLTemplate = domain.MustNewTemplateFromFS(templates, "templates/values.yaml.gotmpl", domain.FormatYAML)

	agentYAMLTemplate      = domain.MustNewTemplateFromFS(templates, "templates/agent/collector.yaml.gotmpl", domain.FormatYAML)
	deploymentYAMLTemplate = domain.MustNewTemplateFromFS(templates, "templates/deployment/collector.yaml.gotmpl", domain.FormatYAML)
)
