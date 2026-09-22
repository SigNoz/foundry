package dockerswarmcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/agent/*.gotmpl
var templates embed.FS

var (
	composeYAMLTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/agent/compose.yaml.gotmpl", domain.FormatYAML)
	agentYAMLTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/agent/agent.yaml.gotmpl", domain.FormatYAML)
)
