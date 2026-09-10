package collectormolding

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

// Per-kind base configs. Both share the same core today (OTLP in, SigNoz
// out); evidence for a kind-level divergence lands in its own file.
var (
	agentConfig      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/agent.yaml.gotmpl", domain.FormatYAML)
	deploymentConfig *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/deployment.yaml.gotmpl", domain.FormatYAML)
)
