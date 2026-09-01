package systemdrpmcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	dropInTemplate    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/foundry.conf.gotmpl", domain.FormatINI)
	agentYAMLTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/agent.yaml.gotmpl", domain.FormatYAML)
)
