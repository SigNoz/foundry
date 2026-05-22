package collectormolding

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

// AgentYAMLTemplate renders the canonical agent OTel collector config. The
// template's . is *collectionagent.CollectorStatus; field names map to
// receivers/processors/exporters/extensions slots and pipeline wiring.
var AgentYAMLTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/agent.yaml.gotmpl", domain.FormatYAML)
