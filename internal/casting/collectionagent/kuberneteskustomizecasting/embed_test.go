package kuberneteskustomizecasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
)

func castingWithKind(t *testing.T, kind collectionagent.CollectorKind) collectionagent.Casting {
	t.Helper()

	config := *collectionagent.Default()
	config.Spec.Collector.Kind = kind

	return config
}

func TestTemplates_RenderValidYAML(t *testing.T) {
	tests := []struct {
		name     string
		template *domain.Template
		kind     collectionagent.CollectorKind
	}{
		{name: "KustomizationTemplate_AgentRendersValidYAML", template: kustomizationTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "KustomizationTemplate_DeploymentRendersValidYAML", template: kustomizationTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "NamespaceTemplate_RendersValidYAML", template: namespaceTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "ServiceaccountTemplate_RendersValidYAML", template: serviceaccountTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "DaemonsetTemplate_RendersValidYAML", template: daemonsetTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "DeploymentTemplate_RendersValidYAML", template: deploymentTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "ServiceTemplate_RendersValidYAML", template: serviceTemplate, kind: collectionagent.CollectorKindDeployment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			material, err := tt.template.Render(castingWithKind(t, tt.kind), "out.yaml")
			assert.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}

func TestEnricherConfigTemplates_RenderEmpty(t *testing.T) {
	tests := []struct {
		name     string
		template *domain.Template
	}{
		{name: "AgentTemplate_RendersEmpty", template: agentYAMLTemplate},
		{name: "DeploymentTemplate_RendersEmpty", template: deploymentYAMLTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			err := tt.template.Execute(buf, nil)

			assert.NoError(t, err)
			assert.NotEmpty(t, buf.String())
		})
	}
}
