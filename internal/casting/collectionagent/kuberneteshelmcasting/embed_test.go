package kuberneteshelmcasting

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// values mirrors the chart keys this casting states. Absent keys unmarshal to
// their zero value, which is what "the template left it to the chart" means.
type values struct {
	FullnameOverride string `json:"fullnameOverride"`
	ClusterName      string `json:"clusterName"`
	Presets          map[string]struct {
		Enabled *bool `json:"enabled"`
	} `json:"presets"`
	OtelAgent      component `json:"otelAgent"`
	OtelDeployment component `json:"otelDeployment"`
}

type component struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
	Image   struct {
		Registry   string `json:"registry"`
		Repository string `json:"repository"`
		Tag        string `json:"tag"`
	} `json:"image"`
	ReplicaCount   int                   `json:"replicaCount"`
	LivenessProbe  struct{ Path string } `json:"livenessProbe"`
	ReadinessProbe struct{ Path string } `json:"readinessProbe"`
	AdditionalEnvs map[string]string     `json:"additionalEnvs"`
	Ports          map[string]struct {
		Enabled       bool   `json:"enabled"`
		ContainerPort int    `json:"containerPort"`
		ServicePort   int    `json:"servicePort"`
		Protocol      string `json:"protocol"`
	} `json:"ports"`
	Config struct {
		Receivers  map[string]any `json:"receivers"`
		Extensions map[string]any `json:"extensions"`
		Service    map[string]any `json:"service"`
	} `json:"config"`
}

// moldedCasting runs the casting's enricher and the collector molding, so the
// values template reads the same merged config a forge does.
func moldedCasting(t *testing.T, kind collectionagent.CollectorKind, env map[string]string) *collectionagent.Casting {
	t.Helper()

	config := collectionagent.Default()
	config.Spec.Collector.Kind = kind
	config.Spec.Collector.Spec.Env = env

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	require.NoError(t, newKubernetesHelmMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindCollector, config))
	require.NoError(t, collectormolding.New(logger).MoldV1Alpha1(ctx, config))
	require.NoError(t, config.MergeStatusIntoSpec())

	return config
}

func defaultEnv() map[string]string {
	return map[string]string{
		"K8S_CLUSTER_NAME":          "production",
		"SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318",
	}
}

func renderValues(t *testing.T, config *collectionagent.Casting) values {
	t.Helper()

	material, err := valuesYAMLTemplate.Render(config, "values.yaml")
	require.NoError(t, err)

	structured, ok := material.(domain.StructuredMaterial)
	require.True(t, ok)

	var rendered values
	require.NoError(t, json.Unmarshal(structured.JSONContents(), &rendered))

	return rendered
}

func TestTemplatesRender(t *testing.T) {
	for name, test := range map[string]struct {
		template *domain.Template
		config   *collectionagent.Casting
	}{
		"AgentValues_Valid":      {valuesYAMLTemplate, moldedCasting(t, collectionagent.CollectorKindAgent, defaultEnv())},
		"DeploymentValues_Valid": {valuesYAMLTemplate, moldedCasting(t, collectionagent.CollectorKindDeployment, defaultEnv())},
		"AgentConfig_Valid":      {agentYAMLTemplate, collectionagent.Default()},
		"DeploymentConfig_Valid": {deploymentYAMLTemplate, collectionagent.Default()},
	} {
		t.Run(name, func(t *testing.T) {
			material, err := test.template.Render(test.config, strings.TrimSuffix(test.template.Name(), ".gotmpl"))

			require.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}

// The enricher's contribution is what the molding merges its base config with,
// so it has to parse as a collector config on its own.
func TestEnricherConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		kind collectionagent.CollectorKind
	}{
		{"Agent_Valid", collectionagent.CollectorKindAgent},
		{"Deployment_Valid", collectionagent.CollectorKindDeployment},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Spec.Collector.Kind = test.kind

			require.NoError(t, newKubernetesHelmMoldingEnricher().EnrichStatus(context.Background(), v1alpha1.MoldingKindCollector, config))

			contributed := config.Spec.Collector.Status.Config.Data[test.kind.ConfigKey()]
			require.NotEmpty(t, contributed)

			var parsed map[string]any
			require.NoError(t, domain.UnmarshalYAML([]byte(contributed), &parsed))

			assert.Contains(t, parsed, "service")
			assert.Contains(t, parsed, "receivers")
		})
	}
}

// Only the casting's own kind is enabled, so the agent and the deployment of one
// casting file install as two releases without overwriting each other's workload.
func TestValuesKindDispatch(t *testing.T) {
	for _, test := range []struct {
		name                      string
		kind                      collectionagent.CollectorKind
		expectedAgentEnabled      bool
		expectedDeploymentEnabled bool
	}{
		{"Agent_Valid", collectionagent.CollectorKindAgent, true, false},
		{"Deployment_Valid", collectionagent.CollectorKindDeployment, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rendered := renderValues(t, moldedCasting(t, test.kind, defaultEnv()))

			assert.Equal(t, "signoz", rendered.FullnameOverride)
			assert.Equal(t, test.expectedAgentEnabled, rendered.OtelAgent.Enabled)
			assert.Equal(t, test.expectedDeploymentEnabled, rendered.OtelDeployment.Enabled)
			assert.Equal(t, "collector-agent", rendered.OtelAgent.Name)
			assert.Equal(t, "collector-deployment", rendered.OtelDeployment.Name)
		})
	}
}

// Every chart preset is off: the collector config foundry pours is the whole
// config, and a preset left on would inject receivers the lock does not carry.
func TestValuesPresets(t *testing.T) {
	rendered := renderValues(t, moldedCasting(t, collectionagent.CollectorKindAgent, defaultEnv()))

	expectedPresets := []string{
		"debugExporter", "otlpExporter", "otlphttpExporter", "logsCollection",
		"hostMetrics", "kubeletMetrics", "kubernetesAttributes", "clusterMetrics",
		"prometheus", "resourceDetection", "k8sEvents",
	}

	for _, name := range expectedPresets {
		preset, ok := rendered.Presets[name]
		require.True(t, ok, "%q must be stated", name)
		require.NotNil(t, preset.Enabled, "%q must state enabled", name)

		assert.False(t, *preset.Enabled, "%q must be off", name)
	}
}

func TestValuesAgent(t *testing.T) {
	rendered := renderValues(t, moldedCasting(t, collectionagent.CollectorKindAgent, defaultEnv()))
	agent := rendered.OtelAgent

	// The chart's probe default is "/"; the base config serves health on /healthz.
	t.Run("Probes_Valid", func(t *testing.T) {
		assert.Equal(t, "/healthz", agent.LivenessProbe.Path)
		assert.Equal(t, "/healthz", agent.ReadinessProbe.Path)
	})

	// K8S_CLUSTER_NAME is the chart's clusterName, not an additional env.
	t.Run("ClusterName_Valid", func(t *testing.T) {
		assert.Equal(t, "production", rendered.ClusterName)
		assert.NotContains(t, agent.AdditionalEnvs, "K8S_CLUSTER_NAME")
		assert.Equal(t, "http://signoz:4318", agent.AdditionalEnvs["SIGNOZ_INGESTION_ENDPOINT"])
	})

	// The chart's otlp receiver would reach the ConfigMap unless nulled out.
	t.Run("ChartDefaultsNulled_Valid", func(t *testing.T) {
		require.Contains(t, agent.Config.Receivers, "otlp")
		assert.Nil(t, agent.Config.Receivers["otlp"])

		assert.Contains(t, agent.Config.Receivers, "otlp/grpc")
		assert.Contains(t, agent.Config.Receivers, "otlp/http")

		require.Contains(t, agent.Config.Extensions, "zpages")
		assert.Nil(t, agent.Config.Extensions["zpages"])
		require.Contains(t, agent.Config.Extensions, "pprof")
		assert.Nil(t, agent.Config.Extensions["pprof"])

		require.Contains(t, agent.Config.Service, "telemetry")
		assert.Nil(t, agent.Config.Service["telemetry"])
	})
}

// Helm deletes a nulled key only when the chart ships it. The chart's deployment
// config declares no receivers, so a nulled otlp would reach the collector as a
// protocol-less receiver and crash it.
func TestValuesDeploymentNoReceiverNull(t *testing.T) {
	deployment := renderValues(t, moldedCasting(t, collectionagent.CollectorKindDeployment, defaultEnv())).OtelDeployment

	assert.NotContains(t, deployment.Config.Receivers, "otlp")

	assert.Contains(t, deployment.Config.Receivers, "otlp/grpc")
	assert.Contains(t, deployment.Config.Receivers, "otlp/http")

	require.Contains(t, deployment.Config.Extensions, "zpages")
	assert.Nil(t, deployment.Config.Extensions["zpages"])
	require.Contains(t, deployment.Config.Service, "telemetry")
	assert.Nil(t, deployment.Config.Service["telemetry"])
}

func TestValuesDeployment(t *testing.T) {
	config := moldedCasting(t, collectionagent.CollectorKindDeployment, defaultEnv())
	config.Spec.Collector.Spec.Cluster.Replicas = v1alpha1.IntPtr(3)

	deployment := renderValues(t, config).OtelDeployment

	// The chart reads replicaCount but ships no default for it.
	t.Run("Replicas_Valid", func(t *testing.T) {
		assert.Equal(t, 3, deployment.ReplicaCount)
	})

	// The chart's deployment ports carry no otlp entries, so the intake the base
	// config listens on would reach no Service.
	t.Run("OTLPPorts_Valid", func(t *testing.T) {
		for name, expectedPort := range map[string]int{"otlp": 4317, "otlp-http": 4318} {
			port, ok := deployment.Ports[name]
			require.True(t, ok, "%q must be stated", name)

			assert.True(t, port.Enabled)
			assert.Equal(t, expectedPort, port.ContainerPort)
			assert.Equal(t, expectedPort, port.ServicePort)
			assert.Equal(t, "TCP", port.Protocol)
		}
	})

	t.Run("Probes_Valid", func(t *testing.T) {
		assert.Equal(t, "/healthz", deployment.LivenessProbe.Path)
		assert.Equal(t, "/healthz", deployment.ReadinessProbe.Path)
	})
}

// The chart takes the image split three ways; an unqualified reference leaves the
// registry to the chart's own default.
func TestValuesImage(t *testing.T) {
	for _, test := range []struct {
		name               string
		image              string
		expectedRegistry   string
		expectedRepository string
		expectedTag        string
	}{
		{"Qualified_Valid", "ghcr.io/signoz/opentelemetry-collector:0.140.0", "ghcr.io", "signoz/opentelemetry-collector", "0.140.0"},
		{"Unqualified_Valid", "otel/opentelemetry-collector-contrib:0.139.0", "", "otel/opentelemetry-collector-contrib", "0.139.0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := moldedCasting(t, collectionagent.CollectorKindAgent, defaultEnv())
			config.Spec.Collector.Spec.Image = test.image

			image := renderValues(t, config).OtelAgent.Image

			assert.Equal(t, test.expectedRegistry, image.Registry)
			assert.Equal(t, test.expectedRepository, image.Repository)
			assert.Equal(t, test.expectedTag, image.Tag)
		})
	}
}

// An unstated cluster name must not render an empty clusterName: the chart falls
// back to global.clusterName only when the key is absent.
func TestValuesClusterNameUnstated(t *testing.T) {
	config := moldedCasting(t, collectionagent.CollectorKindAgent, map[string]string{"SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318"})

	material, err := valuesYAMLTemplate.Render(config, "values.yaml")
	require.NoError(t, err)

	assert.NotContains(t, string(material.FmtContents()), "clusterName")
}
