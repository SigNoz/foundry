package kuberneteskustomizecasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type manifest struct {
	Namespace string `json:"namespace"`
	Metadata  struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Subjects []struct {
		Namespace string `json:"namespace"`
	} `json:"subjects"`
	ConfigMapGenerator []struct {
		Name string `json:"name"`
	} `json:"configMapGenerator"`
	Spec struct {
		Template struct {
			Spec struct {
				Containers []struct {
					Env []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"env"`
				} `json:"containers"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
}

var (
	clusterrolebindingTemplates = map[collectionagent.CollectorKind]*domain.Template{
		collectionagent.CollectorKindAgent:      agentClusterrolebindingTemplate,
		collectionagent.CollectorKindDeployment: deploymentClusterrolebindingTemplate,
	}
	workloadTemplates = map[collectionagent.CollectorKind]*domain.Template{
		collectionagent.CollectorKindAgent:      daemonsetTemplate,
		collectionagent.CollectorKindDeployment: deploymentTemplate,
	}
)

func castingWithKind(t *testing.T, kind collectionagent.CollectorKind) collectionagent.Casting {
	t.Helper()

	config := *collectionagent.Default()
	config.Spec.Collector.Kind = kind

	return config
}

func dataWithKind(t *testing.T, kind collectionagent.CollectorKind) templateData {
	t.Helper()

	return templateDataFor(castingWithKind(t, kind))
}

func render(t *testing.T, tmpl *domain.Template, data templateData) manifest {
	t.Helper()

	material, err := tmpl.Render(data, "out.yaml")
	require.NoError(t, err)

	var rendered manifest
	require.NoError(t, domain.UnmarshalYAML(material.FmtContents(), &rendered))

	return rendered
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
			material, err := tt.template.Render(dataWithKind(t, tt.kind), "out.yaml")
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
			err := tt.template.Execute(buf, dataWithKind(t, tt.kind))
			assert.NoError(t, err)

			var parsed map[string]any
			assert.NoError(t, domain.UnmarshalYAML(buf.Bytes(), &parsed))
			assert.Contains(t, parsed, "service")
		})
	}
}

func TestTemplatesNamespace(t *testing.T) {
	for _, test := range []struct {
		name              string
		annotations       map[string]string
		env               map[string]string
		expectedNamespace string
	}{
		{"Unstated_Valid", nil, nil, "signoz"},
		{"Stated_Valid", map[string]string{collectionagent.KubernetesNamespace.Key: "observability"}, nil, "observability"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := castingWithKind(t, collectionagent.CollectorKindAgent)
			config.Metadata.Annotations = test.annotations
			config.Spec.Collector.Spec.Env = test.env

			data := templateDataFor(config)

			kustomization := render(t, kustomizationTemplate, data)
			assert.Equal(t, test.expectedNamespace, kustomization.Namespace)
			require.Len(t, kustomization.ConfigMapGenerator, 1)
			assert.Equal(t, "signoz-collector-agent", kustomization.ConfigMapGenerator[0].Name)

			assert.Equal(t, test.expectedNamespace, render(t, namespaceTemplate, data).Metadata.Name)

			buf := bytes.NewBuffer(nil)
			require.NoError(t, agentYAMLTemplate.Execute(buf, data))
			assert.Contains(t, buf.String(), "/var/log/pods/"+test.expectedNamespace+"_signoz-collector-*/*/*.log")

			for kind := range workloadTemplates {
				config.Spec.Collector.Kind = kind
				kindData := templateDataFor(config)

				binding := render(t, clusterrolebindingTemplates[kind], kindData)
				require.Len(t, binding.Subjects, 1)
				assert.Equal(t, test.expectedNamespace, binding.Subjects[0].Namespace)
				assert.Equal(t, "signoz-collector-"+kind.String(), binding.Metadata.Name)

				workload := render(t, workloadTemplates[kind], kindData)
				require.Len(t, workload.Spec.Template.Spec.Containers, 1)
			}
		})
	}
}
