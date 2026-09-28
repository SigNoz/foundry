package awskubernetesserverlesskustomizecasting

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type manifest struct {
	ConfigMapGenerator []struct {
		Files []string `json:"files"`
	} `json:"configMapGenerator"`
	Rules []struct {
		Resources []string `json:"resources"`
	} `json:"rules"`
	Spec struct {
		Strategy struct {
			Type string `json:"type"`
		} `json:"strategy"`
		Template struct {
			Spec struct {
				Containers []struct {
					Env []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"env"`
					VolumeMounts []struct {
						Name      string `json:"name"`
						MountPath string `json:"mountPath"`
					} `json:"volumeMounts"`
				} `json:"containers"`
				Volumes []struct {
					Name      string `json:"name"`
					ConfigMap *struct {
						Name string `json:"name"`
					} `json:"configMap"`
				} `json:"volumes"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
}

func castingWithKind(t *testing.T, kind collectionagent.CollectorKind) collectionagent.Casting {
	t.Helper()

	config := *collectionagent.Default()
	config.Spec.Collector.Kind = kind

	return config
}

// Fargate schedules no DaemonSet, and one collector scrapes every node.
func TestForge(t *testing.T) {
	for _, test := range []struct {
		name         string
		kind         collectionagent.CollectorKind
		replicas     *int
		pass         bool
		expectedType int
	}{
		{"Deployment_Valid", collectionagent.CollectorKindDeployment, nil, true, 0},
		{"Agent_Invalid", collectionagent.CollectorKindAgent, nil, false, foundryerrors.TypeUnsupported.ExitCode()},
		{"Sidecar_Invalid", collectionagent.CollectorKindSidecar, nil, false, foundryerrors.TypeUnsupported.ExitCode()},
		{"Replicas_Invalid", collectionagent.CollectorKindDeployment, domain.NewIntPtr(2), false, foundryerrors.TypeUnsupported.ExitCode()},
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
				assert.Equal(t, test.expectedType, foundryerrors.ExitCode(err))

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

			var expectedPaths []string
			for _, file := range []string{"kustomization.yaml", "namespace.yaml", "serviceaccount.yaml", "clusterrole.yaml", "clusterrolebinding.yaml", "service.yaml", "workload.yaml", "kubeconfig.yaml"} {
				expectedPaths = append(expectedPaths, filepath.Join(dir, file))
			}

			assert.ElementsMatch(t, expectedPaths, paths)
		})
	}
}

// The kubeletstats scrapes authenticate through the API-server proxy with the
// kubeconfig the ConfigMap carries beside the collector config.
func TestSurface(t *testing.T) {
	for _, test := range []struct {
		name              string
		template          *domain.Template
		pass              bool
		expectedFiles     []string
		expectedEnv       map[string]string
		expectedMountPath string
		expectedStrategy  string
		expectedResources []string
	}{
		{name: "Kustomization_Valid", template: kustomizationTemplate, pass: true, expectedFiles: []string{"deployment.yaml", "kubeconfig.yaml"}},
		{name: "Workload_Valid", template: deploymentTemplate, pass: true, expectedEnv: map[string]string{"KUBECONFIG": "/conf/kubeconfig.yaml"}, expectedMountPath: "/conf", expectedStrategy: "Recreate"},
		{name: "Clusterrole_Valid", template: deploymentClusterroleTemplate, pass: true, expectedResources: []string{"nodes/stats", "nodes/proxy", "persistentvolumeclaims", "persistentvolumes"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := templateDataFor(castingWithKind(t, collectionagent.CollectorKindDeployment))

			material, err := test.template.Render(data, "out.yaml")
			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			var rendered manifest
			require.NoError(t, domain.UnmarshalYAML(material.FmtContents(), &rendered))

			var files []string
			for _, generator := range rendered.ConfigMapGenerator {
				files = append(files, generator.Files...)
			}

			assert.Subset(t, files, test.expectedFiles)

			var resources []string
			for _, rule := range rendered.Rules {
				resources = append(resources, rule.Resources...)
			}

			assert.Subset(t, resources, test.expectedResources)
			assert.Equal(t, test.expectedStrategy, rendered.Spec.Strategy.Type)

			env := map[string]string{}
			mountPath := ""
			for _, container := range rendered.Spec.Template.Spec.Containers {
				for _, entry := range container.Env {
					if _, ok := test.expectedEnv[entry.Name]; ok {
						env[entry.Name] = entry.Value
					}
				}

				for _, volume := range rendered.Spec.Template.Spec.Volumes {
					if volume.ConfigMap == nil {
						continue
					}

					for _, mount := range container.VolumeMounts {
						if mount.Name == volume.Name {
							mountPath = mount.MountPath
						}
					}
				}
			}

			assert.Equal(t, len(test.expectedEnv), len(env))
			for name, value := range test.expectedEnv {
				assert.Equal(t, value, env[name], name)
			}

			assert.Equal(t, test.expectedMountPath, mountPath)
		})
	}
}

// What EKS Fargate adds rides in the enricher and has to survive the molding's merge.
func TestMoldedConfig(t *testing.T) {
	for _, test := range []struct {
		name               string
		env                map[string]string
		pass               bool
		expectedAttributes map[string]string
	}{
		{
			name: "ClusterName_Valid",
			env:  map[string]string{"K8S_CLUSTER_NAME": "production", "SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318"},
			pass: true,
			expectedAttributes: map[string]string{
				"cloud.provider":   "aws",
				"cloud.platform":   "aws_eks",
				"k8s.cluster.name": "${env:K8S_CLUSTER_NAME}",
			},
		},
		{
			name:               "NoClusterName_Valid",
			env:                map[string]string{},
			pass:               true,
			expectedAttributes: map[string]string{"cloud.provider": "aws", "cloud.platform": "aws_eks"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			config := castingWithKind(t, collectionagent.CollectorKindDeployment)
			config.Spec.Collector.Spec.Env = test.env

			err := newAwsKubernetesServerlessKustomizeMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindCollector, &config)
			if err == nil {
				err = collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(ctx, &config)
			}

			if err == nil {
				err = config.MergeStatusIntoSpec()
			}

			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			contents := config.Spec.Collector.Spec.Config.Data[collectionagent.CollectorKindDeployment.ConfigKey()]
			require.NotEmpty(t, contents)

			merged := domain.MustNewYAMLMaterial([]byte(contents), "deployment.yaml")

			for _, pipeline := range []string{"traces", "metrics", "logs"} {
				processors, err := merged.GetStringSlice("service.pipelines." + pipeline + ".processors")
				require.NoError(t, err)
				assert.Equal(t, []string{"memory_limiter", "resourcedetection", "resource/identity", "k8sattributes", "batch"}, processors, pipeline)
			}

			raw, err := merged.GetBytes("processors.resource/identity.attributes")
			require.NoError(t, err)

			var identity []struct {
				Key    string `json:"key"`
				Value  string `json:"value"`
				Action string `json:"action"`
			}
			require.NoError(t, json.Unmarshal(raw, &identity))

			attributes := map[string]string{}
			for _, attribute := range identity {
				assert.Equal(t, "insert", attribute.Action, attribute.Key)

				attributes[attribute.Key] = attribute.Value
			}

			assert.Equal(t, test.expectedAttributes, attributes)

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

			for path, expected := range map[string]string{
				"extensions.k8s_observer.observe_nodes":                                             "true",
				"extensions.k8s_observer.observe_pods":                                              "false",
				"receivers.receiver_creator.receivers.kubeletstats.rule":                            `type == "k8s.node" && labels["eks.amazonaws.com/compute-type"] == "fargate"`,
				"receivers.receiver_creator.receivers.kubeletstats.config.auth_type":                "kubeConfig",
				"receivers.receiver_creator.receivers.kubeletstats.config.endpoint":                 "`name`",
				"receivers.receiver_creator.receivers.kubeletstats.config.node":                     "`name`",
				"receivers.receiver_creator.receivers.kubeletstats.config.k8s_api_config.auth_type": "serviceAccount",
			} {
				actual, err := merged.GetBytes(path)
				require.NoError(t, err, path)
				assert.Equal(t, expected, string(actual), path)
			}

			for _, path := range []string{
				"receivers.receiver_creator.receivers.kubeletstats.config.insecure_skip_verify",
				"receivers.hostmetrics",
				"receivers.filelog",
				"receivers.kubeletstats",
			} {
				_, err := merged.GetBytes(path)
				assert.Error(t, err, path)
			}
		})
	}
}
