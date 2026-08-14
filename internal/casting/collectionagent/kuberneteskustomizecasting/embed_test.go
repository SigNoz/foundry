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
		{name: "AgentServiceaccountTemplate_RendersValidYAML", template: agentServiceaccountTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "AgentClusterroleTemplate_RendersValidYAML", template: agentClusterroleTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "AgentClusterrolebindingTemplate_RendersValidYAML", template: agentClusterrolebindingTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "AgentServiceTemplate_RendersValidYAML", template: agentServiceTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "DaemonsetTemplate_RendersValidYAML", template: daemonsetTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "DeploymentServiceaccountTemplate_RendersValidYAML", template: deploymentServiceaccountTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentClusterroleTemplate_RendersValidYAML", template: deploymentClusterroleTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentClusterrolebindingTemplate_RendersValidYAML", template: deploymentClusterrolebindingTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentServiceTemplate_RendersValidYAML", template: deploymentServiceTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentTemplate_RendersValidYAML", template: deploymentTemplate, kind: collectionagent.CollectorKindDeployment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			material, err := tt.template.Render(castingWithKind(t, tt.kind), "out.yaml")
			assert.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}

func TestEnricherConfigTemplates_RenderValidYAML(t *testing.T) {
	tests := []struct {
		name     string
		template *domain.Template
		kind     collectionagent.CollectorKind
	}{
		{name: "AgentTemplate_RendersValidYAML", template: agentYAMLTemplate, kind: collectionagent.CollectorKindAgent},
		{name: "DeploymentTemplate_RendersValidYAML", template: deploymentYAMLTemplate, kind: collectionagent.CollectorKindDeployment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			err := tt.template.Execute(buf, castingWithKind(t, tt.kind))
			assert.NoError(t, err)

			var parsed map[string]any
			assert.NoError(t, domain.UnmarshalYAML(buf.Bytes(), &parsed))
			assert.Contains(t, parsed, "service")
		})
	}
}
