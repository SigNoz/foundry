package kuberneteskustomizecasting

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
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

			for _, kind := range collectionagent.CollectorKinds() {
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

// The identity attributes the collector config owns, and the pipeline lists the
// molding's ordered merge produces.
type collectorConfig struct {
	Processors struct {
		Identity *struct {
			Attributes []struct {
				Key    string `json:"key"`
				Action string `json:"action"`
			} `json:"attributes"`
		} `json:"resource/identity"`
	} `json:"processors"`
	Service struct {
		Pipelines map[string]struct {
			Processors []string `json:"processors"`
		} `json:"pipelines"`
	} `json:"service"`
}

type workloadManifest struct {
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

// The molding's ordered list merge puts the enricher's processors before the
// terminal batch, in the order the enricher states them.
func TestMoldedProcessors(t *testing.T) {
	cluster := map[string]string{
		"K8S_CLUSTER_NAME":          "production",
		"SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318",
	}

	for _, test := range []struct {
		name               string
		kind               collectionagent.CollectorKind
		env                map[string]string
		expectedAttributes []string
		expectedProcessors []string
	}{
		{
			"AgentCluster_Valid", collectionagent.CollectorKindAgent, cluster,
			[]string{"host.name", "k8s.node.name", "k8s.cluster.name"},
			[]string{"memory_limiter", "resourcedetection", "resource/identity", "k8sattributes", "batch"},
		},
		{
			"AgentNoCluster_Valid", collectionagent.CollectorKindAgent, map[string]string{},
			[]string{"host.name", "k8s.node.name"},
			[]string{"memory_limiter", "resourcedetection", "resource/identity", "k8sattributes", "batch"},
		},
		{
			"DeploymentCluster_Valid", collectionagent.CollectorKindDeployment, cluster,
			[]string{"k8s.cluster.name"},
			[]string{"memory_limiter", "resourcedetection", "resource/identity", "k8sattributes", "batch"},
		},
		{
			"DeploymentNoCluster_Valid", collectionagent.CollectorKindDeployment, map[string]string{},
			nil,
			[]string{"memory_limiter", "resourcedetection", "k8sattributes", "batch"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			config := collectionagent.Default()
			config.Spec.Collector.Kind = test.kind
			config.Spec.Collector.Spec.Env = test.env

			require.NoError(t, newKubernetesKustomizeMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindCollector, config))
			require.NoError(t, collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(ctx, config))
			require.NoError(t, config.MergeStatusIntoSpec())

			contents := config.Spec.Collector.Spec.Config.Data[test.kind.ConfigKey()]
			require.NotEmpty(t, contents)

			var molded collectorConfig
			require.NoError(t, domain.UnmarshalYAML([]byte(contents), &molded))

			if test.expectedAttributes == nil {
				assert.Nil(t, molded.Processors.Identity)
			} else {
				require.NotNil(t, molded.Processors.Identity)

				var keys []string
				for _, attribute := range molded.Processors.Identity.Attributes {
					assert.Equal(t, "insert", attribute.Action)

					keys = append(keys, attribute.Key)
				}

				assert.Equal(t, test.expectedAttributes, keys)
			}

			for _, pipeline := range []string{"traces", "metrics", "logs"} {
				assert.Equal(t, test.expectedProcessors, molded.Service.Pipelines[pipeline].Processors, "pipeline %q", pipeline)
			}
		})
	}
}

// Identity is the collector config's, so the workload only carries what
// spec.collector.spec.env states.
func TestWorkloadTemplates_ResourceAttributes(t *testing.T) {
	workloadTemplates := map[collectionagent.CollectorKind]*domain.Template{
		collectionagent.CollectorKindAgent:      daemonsetTemplate,
		collectionagent.CollectorKindDeployment: deploymentTemplate,
	}

	for _, test := range []struct {
		name               string
		env                map[string]string
		expectedAttributes []string
	}{
		{"Unstated_Valid", map[string]string{"K8S_CLUSTER_NAME": "production"}, nil},
		{
			"Stated_Valid",
			map[string]string{"OTEL_RESOURCE_ATTRIBUTES": "deployment.environment=production"},
			[]string{"deployment.environment=production"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, kind := range collectionagent.CollectorKinds() {
				config := castingWithKind(t, kind)
				config.Spec.Collector.Spec.Env = test.env

				material, err := workloadTemplates[kind].Render(config, "out.yaml")
				require.NoError(t, err)

				var workload workloadManifest
				require.NoError(t, domain.UnmarshalYAML(material.FmtContents(), &workload))
				require.Len(t, workload.Spec.Template.Spec.Containers, 1)

				var stated []string
				for _, entry := range workload.Spec.Template.Spec.Containers[0].Env {
					if entry.Name == "OTEL_RESOURCE_ATTRIBUTES" {
						stated = append(stated, entry.Value)
					}
				}

				assert.Equal(t, test.expectedAttributes, stated, "workload %q", kind)
			}
		})
	}
}
