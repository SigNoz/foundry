package awskubernetesserverlesskustomizecasting

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
	"github.com/signoz/foundry/internal/pourer"
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
		collectionagent.CollectorKindDeployment: deploymentClusterrolebindingTemplate,
	}
	workloadTemplates = map[collectionagent.CollectorKind]*domain.Template{
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
		{name: "KustomizationTemplate_DeploymentRendersValidYAML", template: kustomizationTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "NamespaceTemplate_RendersValidYAML", template: namespaceTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentServiceaccountTemplate_RendersValidYAML", template: deploymentServiceaccountTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentClusterroleTemplate_RendersValidYAML", template: deploymentClusterroleTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentClusterrolebindingTemplate_RendersValidYAML", template: deploymentClusterrolebindingTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentServiceTemplate_RendersValidYAML", template: deploymentServiceTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentTemplate_RendersValidYAML", template: deploymentTemplate, kind: collectionagent.CollectorKindDeployment},
		{name: "DeploymentKubeconfigTemplate_RendersValidYAML", template: deploymentKubeconfigTemplate, kind: collectionagent.CollectorKindDeployment},
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
			config := castingWithKind(t, collectionagent.CollectorKindDeployment)
			config.Metadata.Annotations = test.annotations
			config.Spec.Collector.Spec.Env = test.env

			data := templateDataFor(config)

			kustomization := render(t, kustomizationTemplate, data)
			assert.Equal(t, test.expectedNamespace, kustomization.Namespace)
			require.Len(t, kustomization.ConfigMapGenerator, 1)
			assert.Equal(t, "signoz-collector-deployment", kustomization.ConfigMapGenerator[0].Name)

			assert.Equal(t, test.expectedNamespace, render(t, namespaceTemplate, data).Metadata.Name)

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

// Fargate schedules no DaemonSet, and one collector scrapes every node.
func TestRefusals(t *testing.T) {
	for _, test := range []struct {
		name     string
		kind     collectionagent.CollectorKind
		replicas *int
		pass     bool
	}{
		{"Deployment_Valid", collectionagent.CollectorKindDeployment, nil, true},
		{"Agent_Invalid", collectionagent.CollectorKindAgent, nil, false},
		{"Sidecar_Invalid", collectionagent.CollectorKindSidecar, nil, false},
		{"Replicas_Invalid", collectionagent.CollectorKindDeployment, domain.NewIntPtr(2), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := castingWithKind(t, test.kind)
			config.Spec.Collector.Spec.Cluster.Replicas = test.replicas

			p := pourer.New("collectionagent")

			err := newAwsKubernetesServerlessKustomizeMoldingEnricher().EnrichStatus(context.Background(), v1alpha1.MoldingKindCollector, &config)
			if err == nil {
				err = New(slog.New(slog.DiscardHandler)).Forge(context.Background(), config, p)
			}

			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			materials, err := p.Pour()
			require.NoError(t, err)

			var paths []string
			for _, material := range materials {
				paths = append(paths, material.Path())
			}

			dir := filepath.Join("collectionagent", filepath.Dir(test.kind.ConfigKey()))
			for _, file := range []string{"kustomization.yaml", "namespace.yaml", "serviceaccount.yaml", "clusterrole.yaml", "clusterrolebinding.yaml", "service.yaml", "workload.yaml", "kubeconfig.yaml"} {
				assert.Contains(t, paths, filepath.Join(dir, file))
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
				Value  string `json:"value"`
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
		expectedAttributes map[string]string
	}{
		{
			"DeploymentCluster_Valid", collectionagent.CollectorKindDeployment, cluster,
			map[string]string{"cloud.provider": "aws", "cloud.platform": "aws_eks", "k8s.cluster.name": "${env:K8S_CLUSTER_NAME}"},
		},
		{
			"DeploymentNoCluster_Valid", collectionagent.CollectorKindDeployment, map[string]string{},
			map[string]string{"cloud.provider": "aws", "cloud.platform": "aws_eks"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			config := collectionagent.Default()
			config.Spec.Collector.Kind = test.kind
			config.Spec.Collector.Spec.Env = test.env

			require.NoError(t, newAwsKubernetesServerlessKustomizeMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindCollector, config))
			require.NoError(t, collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(ctx, config))
			require.NoError(t, config.MergeStatusIntoSpec())

			contents := config.Spec.Collector.Spec.Config.Data[test.kind.ConfigKey()]
			require.NotEmpty(t, contents)

			var molded collectorConfig
			require.NoError(t, domain.UnmarshalYAML([]byte(contents), &molded))

			require.NotNil(t, molded.Processors.Identity)

			attributes := map[string]string{}
			for _, attribute := range molded.Processors.Identity.Attributes {
				assert.Equal(t, "insert", attribute.Action)

				attributes[attribute.Key] = attribute.Value
			}

			assert.Equal(t, test.expectedAttributes, attributes)

			for _, pipeline := range []string{"traces", "metrics", "logs"} {
				assert.Equal(t, []string{"memory_limiter", "resourcedetection", "resource/identity", "k8sattributes", "batch"}, molded.Service.Pipelines[pipeline].Processors, "pipeline %q", pipeline)
			}
		})
	}
}

// What EKS Fargate adds rides in the enricher and has to survive the molding's merge.
func TestDeploymentConfig(t *testing.T) {
	ctx := context.Background()

	config := collectionagent.Default()
	config.Spec.Collector.Kind = collectionagent.CollectorKindDeployment

	require.NoError(t, newAwsKubernetesServerlessKustomizeMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindCollector, config))
	require.NoError(t, collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(ctx, config))

	merged := domain.MustNewYAMLMaterial([]byte(config.Spec.Collector.Status.Config.Data[collectionagent.CollectorKindDeployment.ConfigKey()]), "deployment.yaml")

	detectors, err := merged.GetStringSlice("processors.resourcedetection.detectors")
	require.NoError(t, err)
	assert.Equal(t, []string{"env"}, detectors)

	for path, expected := range map[string]string{
		"service.extensions":                  "k8s_observer",
		"service.pipelines.metrics.receivers": "receiver_creator",
	} {
		actual, err := merged.GetStringSlice(path)
		require.NoError(t, err)
		assert.Contains(t, actual, expected, path)
	}

	for _, path := range []string{"receivers.hostmetrics", "receivers.filelog", "receivers.kubeletstats"} {
		_, err := merged.GetBytes(path)
		assert.Error(t, err, path)
	}
}

// Identity is the collector config's, so the workload only carries what
// spec.collector.spec.env states.
func TestWorkloadTemplates_ResourceAttributes(t *testing.T) {
	workloadTemplates := map[collectionagent.CollectorKind]*domain.Template{
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
			for kind := range workloadTemplates {
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
