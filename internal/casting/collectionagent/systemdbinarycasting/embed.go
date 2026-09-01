package systemdbinarycasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	serviceTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/collector-agent.service.gotmpl", domain.FormatINI)
	agentYAMLTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/agent.yaml.gotmpl", domain.FormatYAML)
)
