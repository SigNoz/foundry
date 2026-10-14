package gcpkubernetesautohelmcasting

import (
	"context"
	"log/slog"
	"strings"
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

func moldedCasting(t *testing.T, kind collectionagent.CollectorKind) *collectionagent.Casting {
	t.Helper()

	config := collectionagent.Default()
	config.Spec.Collector.Kind = kind
	config.Spec.Collector.Spec.Env = defaultEnv()

	ctx := context.Background()

	require.NoError(t, newGcpKubernetesAutoHelmMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindCollector, config))
	require.NoError(t, collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(ctx, config))
	require.NoError(t, config.MergeStatusIntoSpec())

	return config
}

func defaultEnv() map[string]string {
	return map[string]string{
		"K8S_CLUSTER_NAME":          "production",
		"SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318",
	}
}

func TestTemplatesRender(t *testing.T) {
	for name, test := range map[string]struct {
		template *domain.Template
		config   *collectionagent.Casting
		pass     bool
	}{
		"AgentValues_Valid":      {valuesYAMLTemplate, moldedCasting(t, collectionagent.CollectorKindAgent), true},
		"DeploymentValues_Valid": {valuesYAMLTemplate, moldedCasting(t, collectionagent.CollectorKindDeployment), true},
		"AgentConfig_Valid":      {agentYAMLTemplate, collectionagent.Default(), true},
		"DeploymentConfig_Valid": {deploymentYAMLTemplate, collectionagent.Default(), true},
	} {
		t.Run(name, func(t *testing.T) {
			material, err := test.template.Render(templateDataFor(*test.config), strings.TrimSuffix(test.template.Name(), ".gotmpl"))

			if test.pass {
				require.NoError(t, err)
				assert.NotEmpty(t, material.FmtContents())

				return
			}

			require.Error(t, err)
		})
	}
}

// On gcp/autogke the chart mounts no host root; a stated hostfs volume would bring one back.
func TestForge(t *testing.T) {
	for _, test := range []struct {
		name              string
		kind              collectionagent.CollectorKind
		controller        string
		pass              bool
		expectedErrorType int
		expectedPath      string
	}{
		{"Agent_Valid", collectionagent.CollectorKindAgent, "", true, 0, "collectionagent/collector/agent/values.yaml"},
		{"Deployment_Valid", collectionagent.CollectorKindDeployment, "", true, 0, "collectionagent/collector/deployment/values.yaml"},
		{"OpenTelemetryOperator_Invalid", collectionagent.CollectorKindAgent, collectionagent.CollectorControllerOpenTelemetryOperator, false, foundryerrors.TypeInvalidInput.ExitCode(), ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := moldedCasting(t, test.kind)

			if test.controller != "" {
				config.Metadata.Annotations = map[string]string{collectionagent.KubernetesCollectorController.Key: test.controller}
			}

			p := pourer.New("collectionagent")
			err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *config, p)

			if !test.pass {
				require.Error(t, err)
				assert.Equal(t, test.expectedErrorType, foundryerrors.ExitCode(err))

				return
			}

			require.NoError(t, err)

			materials, err := p.Pour()
			require.NoError(t, err)
			require.Len(t, materials, 1)

			assert.Equal(t, test.expectedPath, materials[0].Path())

			var rendered struct {
				Global struct {
					Cloud string `json:"cloud"`
				} `json:"global"`
				OtelAgent map[string]any `json:"otelAgent"`
			}
			require.NoError(t, domain.UnmarshalYAML(materials[0].FmtContents(), &rendered))

			assert.Equal(t, "gcp/autogke", rendered.Global.Cloud)
			assert.NotContains(t, rendered.OtelAgent, "extraVolumes")
			assert.NotContains(t, rendered.OtelAgent, "extraVolumeMounts")
		})
	}
}

// What Autopilot changes rides in the enricher and has to survive the molding's merge.
func TestMoldedConfig(t *testing.T) {
	for _, test := range []struct {
		name                    string
		kind                    collectionagent.CollectorKind
		pass                    bool
		expectedErrorType       int
		expectedDetectors       []string
		expectedGCPDisabled     []string
		expectedAbsent          []string
		expectedKubeletDisabled []string
	}{
		{
			name:                "Agent_Valid",
			kind:                collectionagent.CollectorKindAgent,
			pass:                true,
			expectedDetectors:   []string{"k8snode", "env", "gcp", "system"},
			expectedGCPDisabled: []string{"host.id", "host.name", "k8s.cluster.name"},
			expectedAbsent:      []string{"receivers.hostmetrics.root_path", "receivers.kubeletstats.extra_metadata_labels"},
			expectedKubeletDisabled: []string{
				"k8s.container.cpu_limit_utilization",
				"k8s.container.cpu_request_utilization",
				"k8s.container.memory_limit_utilization",
				"k8s.container.memory_request_utilization",
				"k8s.pod.cpu_limit_utilization",
				"k8s.pod.cpu_request_utilization",
				"k8s.pod.memory_limit_utilization",
				"k8s.pod.memory_request_utilization",
			},
		},
		{
			name:              "Deployment_Valid",
			kind:              collectionagent.CollectorKindDeployment,
			pass:              true,
			expectedDetectors: []string{"env", "gcp"},
			expectedGCPDisabled: []string{
				"gcp.gce.instance.hostname",
				"gcp.gce.instance.name",
				"host.id",
				"host.name",
				"host.type",
				"k8s.cluster.name",
			},
		},
		{
			name:              "Sidecar_Invalid",
			kind:              collectionagent.CollectorKindSidecar,
			pass:              false,
			expectedErrorType: foundryerrors.TypeUnsupported.ExitCode(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			config := collectionagent.Default()
			config.Spec.Collector.Kind = test.kind
			config.Spec.Collector.Spec.Env = defaultEnv()

			err := newGcpKubernetesAutoHelmMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindCollector, config)
			if err == nil {
				err = collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(ctx, config)
			}

			if err == nil {
				err = config.MergeStatusIntoSpec()
			}

			if !test.pass {
				require.Error(t, err)
				assert.Equal(t, test.expectedErrorType, foundryerrors.ExitCode(err))

				return
			}

			require.NoError(t, err)

			contents := config.Spec.Collector.Spec.Config.Data[test.kind.ConfigKey()]
			require.NotEmpty(t, contents)

			var merged struct {
				Receivers struct {
					Kubeletstats struct {
						Metrics map[string]struct {
							Enabled bool `json:"enabled"`
						} `json:"metrics"`
					} `json:"kubeletstats"`
				} `json:"receivers"`
				Processors struct {
					Resourcedetection struct {
						Detectors []string `json:"detectors"`
						GCP       struct {
							ResourceAttributes map[string]struct {
								Enabled bool `json:"enabled"`
							} `json:"resource_attributes"`
						} `json:"gcp"`
					} `json:"resourcedetection"`
				} `json:"processors"`
			}
			require.NoError(t, domain.UnmarshalYAML([]byte(contents), &merged))

			detection := merged.Processors.Resourcedetection
			assert.Equal(t, test.expectedDetectors, detection.Detectors)

			gcpDisabled := []string{}
			for attribute, setting := range detection.GCP.ResourceAttributes {
				if !setting.Enabled {
					gcpDisabled = append(gcpDisabled, attribute)
				}
			}

			assert.ElementsMatch(t, test.expectedGCPDisabled, gcpDisabled)

			material := domain.MustNewYAMLMaterial([]byte(contents), "collector.yaml")
			for _, path := range test.expectedAbsent {
				_, err := material.GetBytes(path)
				assert.Error(t, err, path)
			}

			kubeletDisabled := []string{}
			for metric, setting := range merged.Receivers.Kubeletstats.Metrics {
				if !setting.Enabled {
					kubeletDisabled = append(kubeletDisabled, metric)
				}
			}

			assert.ElementsMatch(t, test.expectedKubeletDisabled, kubeletDisabled)
		})
	}
}
