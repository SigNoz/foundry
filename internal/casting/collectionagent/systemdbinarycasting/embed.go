package systemdbinarycasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/agent/*.gotmpl
var templates embed.FS

var (
	agentYAMLTemplate        = domain.MustNewTemplateFromFS(templates, "templates/agent/collector.yaml.gotmpl", domain.FormatYAML)
	agentUnitTemplate        = domain.MustNewTemplateFromFS(templates, "templates/agent/unit.service.gotmpl", domain.FormatINI)
	agentEnvironmentTemplate = domain.MustNewTemplateFromFS(templates, "templates/agent/environment.conf.gotmpl", domain.FormatText)
	agentSysusersTemplate    = domain.MustNewTemplateFromFS(templates, "templates/agent/sysusers.conf.gotmpl", domain.FormatText)
)
