package kuberneteshelmcasting

import (
	"context"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

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

// Stated mutations run before the enricher, which is where a user's spec fields reach it.
func moldedCasting(t *testing.T, kind collectionagent.CollectorKind, env map[string]string, stated ...func(*collectionagent.Casting)) *collectionagent.Casting {
	t.Helper()

	config := collectionagent.Default()
	config.Spec.Collector.Kind = kind
	config.Spec.Collector.Spec.Env = env

	for _, mutate := range stated {
		mutate(config)
	}

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

func renderValuesYAML(t *testing.T, config *collectionagent.Casting) []byte {
	t.Helper()

	material, err := valuesYAMLTemplate.Render(templateDataFor(*config), "values.yaml")
	require.NoError(t, err)

	return material.FmtContents()
}

func renderValues(t *testing.T, config *collectionagent.Casting) values {
	t.Helper()

	var rendered values
	require.NoError(t, domain.UnmarshalYAML(renderValuesYAML(t, config), &rendered))

	return rendered
}

func foundryConfig(t *testing.T, config *collectionagent.Casting) map[string]any {
	t.Helper()

	contents := config.Spec.Collector.Spec.Config.Data[config.Spec.Collector.Kind.ConfigKey()]
	require.NotEmpty(t, contents)

	var parsed map[string]any
	require.NoError(t, domain.UnmarshalYAML([]byte(contents), &parsed))

	return parsed
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
			material, err := test.template.Render(templateDataFor(*test.config), strings.TrimSuffix(test.template.Name(), ".gotmpl"))

			require.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}

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

// A preset left on would inject receivers the lock does not carry.
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
	for _, test := range []struct {
		name                string
		env                 map[string]string
		expectedClusterName string
	}{
		{"ClusterName_Valid", defaultEnv(), "production"},
		// The chart falls back to global.clusterName only when the key is absent.
		{"ClusterNameUnstated_Valid", map[string]string{"SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318"}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := moldedCasting(t, collectionagent.CollectorKindAgent, test.env)

			rendered := renderValuesYAML(t, config)
			assert.Equal(t, test.expectedClusterName != "", strings.Contains(string(rendered), "clusterName"))

			var parsed values
			require.NoError(t, domain.UnmarshalYAML(rendered, &parsed))
			assert.Equal(t, test.expectedClusterName, parsed.ClusterName)

			agent := parsed.OtelAgent

			// The chart's probe default is "/"; the base config serves health on /healthz.
			assert.Equal(t, "/healthz", agent.LivenessProbe.Path)
			assert.Equal(t, "/healthz", agent.ReadinessProbe.Path)

			// K8S_CLUSTER_NAME is the chart's clusterName, not an additional env.
			assert.NotContains(t, agent.AdditionalEnvs, "K8S_CLUSTER_NAME")
			assert.Equal(t, "http://signoz:4318", agent.AdditionalEnvs["SIGNOZ_INGESTION_ENDPOINT"])

			// The chart's otlp receiver would reach the ConfigMap unless nulled out.
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
}

// Helm deletes a nulled key only when the chart ships it, and the chart's
// deployment config ships no receivers: a nulled otlp would crash the collector.
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

	// The chart's deployment ports carry no otlp entries, so the intake reaches no Service.
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

// The chart takes the image split three ways; an unqualified reference leaves the registry to it.
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

const agentConfigOverride = `receivers:
  filelog/custom:
    include:
      - /var/log/custom/*.log
exporters:
  otlphttp/signoz:
    headers:
      signoz-ingestion-key: ${env:SIGNOZ_INGESTION_KEY}
`

var (
	agentOnly      = []collectionagent.CollectorKind{collectionagent.CollectorKindAgent}
	deploymentOnly = []collectionagent.CollectorKind{collectionagent.CollectorKindDeployment}
	cloudCases     = []string{"CloudAWS", "CloudAzure", "CloudGCP", "CloudAutoGKE"}
)

// A nil kinds means both; passthrough admits keys only foundry states, which
// with every preset off is the premise of the casting.
var chartCases = []struct {
	name        string
	values      map[string]any
	kinds       []collectionagent.CollectorKind
	passthrough bool
}{
	{name: "Defaults", values: map[string]any{}},
	{name: "CloudAWS", values: map[string]any{"global": map[string]any{"cloud": "aws"}}},
	{name: "CloudAzure", values: map[string]any{"global": map[string]any{"cloud": "azure"}}},
	{name: "CloudGCP", values: map[string]any{"global": map[string]any{"cloud": "gcp"}}},
	{name: "CloudAutoGKE", values: map[string]any{"global": map[string]any{"cloud": "gcp/autogke"}}},
	{name: "DeploymentEnvironment", values: map[string]any{"global": map[string]any{"deploymentEnvironment": "staging"}}},
	{name: "TLS", values: map[string]any{"insecureSkipVerify": true, "otelTlsSecrets": map[string]any{"enabled": true, "ca": "x"}}},
	{name: "DebugExporter", values: map[string]any{"presets": map[string]any{"debugExporter": map[string]any{"enabled": true}}}},
	{name: "OTLPExporter", values: map[string]any{"presets": map[string]any{
		"otlpExporter":     map[string]any{"enabled": true},
		"otlphttpExporter": map[string]any{"enabled": false},
	}}},
	{name: "Prometheus", kinds: deploymentOnly, values: map[string]any{"presets": map[string]any{"prometheus": map[string]any{"enabled": true}}}},
	{name: "K8sEventsNamespaces", kinds: deploymentOnly, values: map[string]any{"presets": map[string]any{"k8sEvents": map[string]any{"namespaces": []any{"a"}}}}},
	{name: "LogsWhitelist", kinds: agentOnly, values: map[string]any{"presets": map[string]any{"logsCollection": map[string]any{"whitelist": map[string]any{"enabled": true}}}}},
	{name: "LogsMultiline", kinds: agentOnly, values: map[string]any{"presets": map[string]any{"logsCollection": map[string]any{"multiline": map[string]any{"line_start_pattern": "^[0-9]{4}"}}}}},
	{name: "SelfTelemetry", values: map[string]any{"presets": map[string]any{"selfTelemetry": map[string]any{
		"traces":  map[string]any{"enabled": true},
		"metrics": map[string]any{"enabled": true},
		"logs":    map[string]any{"enabled": true},
	}}}},
	{name: "PresetsOff", passthrough: true, values: map[string]any{"presets": map[string]any{
		"debugExporter":        map[string]any{"enabled": false},
		"otlpExporter":         map[string]any{"enabled": false},
		"otlphttpExporter":     map[string]any{"enabled": false},
		"logsCollection":       map[string]any{"enabled": false},
		"hostMetrics":          map[string]any{"enabled": false},
		"kubeletMetrics":       map[string]any{"enabled": false},
		"kubernetesAttributes": map[string]any{"enabled": false},
		"clusterMetrics":       map[string]any{"enabled": false},
		"prometheus":           map[string]any{"enabled": false},
		"resourceDetection":    map[string]any{"enabled": false},
		"k8sEvents":            map[string]any{"enabled": false},
	}}},
}

// The reason a divergence is allowed, keyed by the path prefix it covers: the
// prefix itself or a path under it, so receivers.otlp does not swallow
// receivers.otlp/grpc. A nil cases or kinds means every case or both kinds.
var chartAllowlist = []struct {
	prefix string
	reason string
	cases  []string
	kinds  []collectionagent.CollectorKind
}{
	{"exporters", "divergence 1: foundry exports through otlphttp/signoz and SIGNOZ_INGESTION_ENDPOINT", nil, nil},
	{"extensions", "divergences 2 and 6: no pprof or zpages, and health_check serves /healthz", nil, nil},
	{"service.extensions", "divergence 2: no pprof or zpages in the base config", nil, nil},
	{"processors.batch", "divergence 3: the base config's 1000/2048/10s wins", nil, nil},
	{"processors.memory_limiter", "divergence 4: base-owned, the chart has none", nil, nil},
	{"receivers.otlp", "divergence 5: the chart's single 4MiB intake is split", nil, nil},
	{"receivers.otlp/grpc", "divergence 5: the chart's single 4MiB intake is split", nil, nil},
	{"receivers.otlp/http", "divergence 5: the chart's single 4MiB intake is split", nil, nil},
	{"service.pipelines.traces.receivers", "divergence 5: the chart's single 4MiB intake is split", nil, nil},
	{"service.pipelines.metrics.receivers", "divergence 5: the chart's single 4MiB intake is split", nil, nil},
	{"service.pipelines.logs.receivers", "divergence 5: the chart's single 4MiB intake is split", nil, nil},
	{"service.pipelines.traces.processors", "divergence 9: foundry states its own processor order", nil, nil},
	{"service.pipelines.metrics.processors", "divergence 9: foundry states its own processor order", nil, nil},
	{"service.pipelines.logs.processors", "divergence 9: foundry states its own processor order", nil, nil},
	{"service.pipelines.traces.exporters", "divergence 1: foundry exports through otlphttp/signoz", nil, nil},
	{"service.pipelines.metrics.exporters", "divergence 1: foundry exports through otlphttp/signoz", nil, nil},
	{"service.pipelines.logs.exporters", "divergence 1: foundry exports through otlphttp/signoz", nil, nil},
	{"service.pipelines.metrics", "divergence 5: foundry's deployment keeps one metrics pipeline", nil, deploymentOnly},
	{"service.pipelines.metrics/internal", "divergence 5: the chart splits internal from scraper", nil, deploymentOnly},
	{"service.pipelines.traces", "divergence 5: the chart's deployment has no traces pipeline", nil, deploymentOnly},
	{"receivers.filelog/k8s.exclude", "divergence 8: foundry excludes its own pods by namespace", nil, agentOnly},
	{"service.telemetry", "2026-09-12: the chart's logs.encoding json is nulled away", nil, nil},
	{"processors.k8sattributes.extract.annotations", "divergence 12: the chart emits empty extract lists", nil, agentOnly},
	{"processors.k8sattributes.extract.labels", "divergence 12: the chart emits empty extract lists", nil, agentOnly},
	{"processors.resource/identity", "divergence 13: identity attributes are config-owned", nil, nil},
	{"processors.resourcedetection.detectors", "cloud deltas parked, vanilla first", cloudCases, nil},
	{"processors.resourcedetection.ec2", "cloud deltas parked, vanilla first", []string{"CloudAWS"}, nil},
	{"processors.resourcedetection.gcp", "cloud deltas parked, vanilla first", []string{"CloudGCP", "CloudAutoGKE"}, nil},
	{"processors.resourcedetection.azure", "cloud deltas parked, vanilla first", []string{"CloudAzure"}, nil},
	{"receivers.hostmetrics.root_path", "cloud deltas parked, vanilla first", []string{"CloudAutoGKE"}, agentOnly},
	{"receivers.kubeletstats.extra_metadata_labels", "cloud deltas parked, vanilla first", []string{"CloudAutoGKE"}, agentOnly},
	{"receivers.kubeletstats.metrics", "cloud deltas parked, vanilla first", []string{"CloudAutoGKE"}, agentOnly},
	{"processors.resource/deployenv", "attribution contract 3: the environment travels on OTEL_RESOURCE_ATTRIBUTES", []string{"DeploymentEnvironment"}, nil},
	{"receivers.prometheus/scraper", "opt-in chart feature, foundry's channel is spec.config.data", []string{"Prometheus"}, nil},
	{"service.pipelines.metrics/scraper", "opt-in chart feature, foundry's channel is spec.config.data", []string{"Prometheus"}, nil},
	{"receivers.k8s_events.namespaces", "opt-in chart feature, foundry's channel is spec.config.data", []string{"K8sEventsNamespaces"}, nil},
	{"receivers.filelog/k8s.include", "opt-in chart feature, foundry's channel is spec.config.data", []string{"LogsWhitelist"}, nil},
	{"receivers.filelog/k8s.multiline", "opt-in chart feature, foundry's channel is spec.config.data", []string{"LogsMultiline"}, nil},
	{"receivers.filelog/self_logs", "opt-in chart feature, foundry's channel is spec.config.data", []string{"SelfTelemetry"}, nil},
	{"processors.filter/non_error_logs", "opt-in chart feature, foundry's channel is spec.config.data", []string{"SelfTelemetry"}, nil},
	{"service.pipelines.logs/self_logs", "opt-in chart feature, foundry's channel is spec.config.data", []string{"SelfTelemetry"}, nil},
}

// TestChart runs against a local k8s-infra checkout: the annotations leave the
// chart version unpinned, so nothing here is vendored or fetched.
func TestChart(t *testing.T) {
	dir := os.Getenv("FOUNDRY_K8S_INFRA_CHART")
	if dir == "" {
		t.Skip("set FOUNDRY_K8S_INFRA_CHART to a local k8s-infra chart directory")
	}

	chrt, err := loader.LoadDir(dir)
	require.NoError(t, err)

	userEnv := defaultEnv()
	userEnv["OTEL_RESOURCE_ATTRIBUTES"] = "deployment.environment=production"

	for _, test := range []struct {
		name   string
		config *collectionagent.Casting
	}{
		{"Fidelity_Agent_Equal", moldedCasting(t, collectionagent.CollectorKindAgent, defaultEnv())},
		{"Fidelity_Deployment_Equal", moldedCasting(t, collectionagent.CollectorKindDeployment, defaultEnv())},
		{"Fidelity_AgentUserResourceAttributes_Equal", moldedCasting(t, collectionagent.CollectorKindAgent, userEnv)},
		{
			"Fidelity_AgentConfigOverride_Equal",
			moldedCasting(t, collectionagent.CollectorKindAgent, defaultEnv(), func(config *collectionagent.Casting) {
				config.Spec.Collector.Spec.Config.Set(collectionagent.CollectorKindAgent.ConfigKey(), []byte(agentConfigOverride))
			}),
		},
		{
			"Fidelity_DeploymentReplicas_Equal",
			moldedCasting(t, collectionagent.CollectorKindDeployment, defaultEnv(), func(config *collectionagent.Casting) {
				config.Spec.Collector.Spec.Cluster.Replicas = v1alpha1.IntPtr(2)
			}),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			vals := map[string]any{}
			require.NoError(t, domain.UnmarshalYAML(renderValuesYAML(t, test.config), &vals))

			kind := test.config.Spec.Collector.Kind
			manifests := renderChart(t, chrt, vals, releaseName(*test.config), namespace(*test.config))

			assert.Equal(t, foundryConfig(t, test.config), chartConfig(t, manifests, chrt, kind))

			for _, other := range collectionagent.CollectorKinds() {
				if other == kind {
					continue
				}

				assert.Empty(t, strings.TrimSpace(manifests[configMapTemplate(chrt, other)]), "the chart renders a %s ConfigMap too", other)
			}
		})
	}

	molded := map[collectionagent.CollectorKind]map[string]any{}
	for _, kind := range collectionagent.CollectorKinds() {
		molded[kind] = foundryConfig(t, moldedCasting(t, kind, defaultEnv()))
	}

	for _, test := range chartCases {
		for _, kind := range collectionagent.CollectorKinds() {
			if len(test.kinds) > 0 && !slices.Contains(test.kinds, kind) {
				continue
			}

			t.Run("Parity_"+test.name+"_"+kind.String()+"_Explained", func(t *testing.T) {
				manifests := renderChart(t, chrt, test.values, "signoz", "signoz")

				diffs := map[string]string{}
				diffConfig(chartConfig(t, manifests, chrt, kind), molded[kind], "", diffs)

				residual := []string{}
				for path, side := range diffs {
					if test.passthrough && side == "foundry" {
						continue
					}

					if explained(path, test.name, kind) == "" {
						residual = append(residual, path+" ("+side+")")
					}
				}

				slices.Sort(residual)
				assert.Empty(t, residual, "unexplained divergences from the chart")
			})
		}
	}

	// Defaults only: an opt-in case injects components foundry does not mirror.
	manifests := renderChart(t, chrt, map[string]any{}, "signoz", "signoz")
	renames := map[string][]string{"otlp": {"otlp/grpc", "otlp/http"}, "otlphttp": {"otlphttp/signoz"}}

	for _, kind := range collectionagent.CollectorKinds() {
		t.Run("Membership_"+kind.String()+"_Valid", func(t *testing.T) {
			mine := pipelines(t, molded[kind])

			for name, declared := range pipelines(t, chartConfig(t, manifests, chrt, kind)) {
				pipeline, ok := declared.(map[string]any)
				require.True(t, ok)

				target := name
				if target == "metrics/internal" {
					target = "metrics"
				}

				stated, ok := mine[target].(map[string]any)
				require.True(t, ok, "foundry states no %q pipeline", target)

				for _, class := range []string{"receivers", "processors", "exporters"} {
					foundrys := components(t, stated, class)

					for _, chartComponent := range components(t, pipeline, class) {
						expected, renamed := renames[chartComponent]
						if !renamed {
							expected = []string{chartComponent}
						}

						for _, component := range expected {
							assert.Contains(t, foundrys, component, "%s %q in the chart's %q pipeline", class, chartComponent, name)
						}
					}
				}
			}
		})
	}
}

func explained(path, name string, kind collectionagent.CollectorKind) string {
	for _, entry := range chartAllowlist {
		if entry.prefix != path && !strings.HasPrefix(path, entry.prefix+".") {
			continue
		}

		if len(entry.cases) > 0 && !slices.Contains(entry.cases, name) {
			continue
		}

		if len(entry.kinds) > 0 && !slices.Contains(entry.kinds, kind) {
			continue
		}

		return entry.reason
	}

	return ""
}

func diffConfig(chartSide, foundrySide any, path string, diffs map[string]string) {
	chartMap, chartIsMap := chartSide.(map[string]any)
	foundryMap, foundryIsMap := foundrySide.(map[string]any)

	if !chartIsMap || !foundryIsMap {
		if !reflect.DeepEqual(chartSide, foundrySide) {
			diffs[path] = "both"
		}

		return
	}

	for key, chartValue := range chartMap {
		foundryValue, stated := foundryMap[key]
		if !stated {
			diffs[joinPath(path, key)] = "chart"

			continue
		}

		diffConfig(chartValue, foundryValue, joinPath(path, key), diffs)
	}

	for key := range foundryMap {
		if _, stated := chartMap[key]; !stated {
			diffs[joinPath(path, key)] = "foundry"
		}
	}
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}

	return path + "." + key
}

func renderChart(t *testing.T, chrt *chart.Chart, vals map[string]any, release, ns string) map[string]string {
	t.Helper()

	rendered, err := chartutil.ToRenderValues(chrt, vals,
		chartutil.ReleaseOptions{Name: release, Namespace: ns, Revision: 1, IsInstall: true},
		chartutil.DefaultCapabilities)
	require.NoError(t, err)

	manifests, err := engine.Render(chrt, rendered)
	require.NoError(t, err)

	return manifests
}

func configMapTemplate(chrt *chart.Chart, kind collectionagent.CollectorKind) string {
	return chrt.Name() + "/templates/otel-" + kind.String() + "/configmap.yaml"
}

func chartConfig(t *testing.T, manifests map[string]string, chrt *chart.Chart, kind collectionagent.CollectorKind) map[string]any {
	t.Helper()

	var configMap struct {
		Data map[string]string `json:"data"`
	}
	require.NoError(t, domain.UnmarshalYAML([]byte(manifests[configMapTemplate(chrt, kind)]), &configMap))

	contents, ok := configMap.Data["otel-"+kind.String()+"-config.yaml"]
	require.True(t, ok, "the chart renders no %s ConfigMap", kind)

	var parsed map[string]any
	require.NoError(t, domain.UnmarshalYAML([]byte(contents), &parsed))

	return parsed
}

func pipelines(t *testing.T, config map[string]any) map[string]any {
	t.Helper()

	service, ok := config["service"].(map[string]any)
	require.True(t, ok)

	declared, ok := service["pipelines"].(map[string]any)
	require.True(t, ok)

	return declared
}

func components(t *testing.T, pipeline map[string]any, class string) []string {
	t.Helper()

	entries, _ := pipeline[class].([]any)

	names := []string{}
	for _, entry := range entries {
		name, ok := entry.(string)
		require.True(t, ok)

		names = append(names, name)
	}

	return names
}
