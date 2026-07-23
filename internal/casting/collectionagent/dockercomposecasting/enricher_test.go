package dockercomposecasting

import (
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

// moldedAgentConfig enriches and molds a default agent casting, returning the
// resolved collector config.
func moldedAgentConfig(t *testing.T) map[string]any {
	t.Helper()

	c := collectionagent.Default()
	enricher := newDockerComposeMoldingEnricher()
	require.NoError(t, enricher.EnrichStatus(context.Background(), v1alpha1.MoldingKindCollector, c))
	require.NoError(t, collectormolding.New(slog.Default()).MoldV1Alpha1(context.Background(), c))

	var config map[string]any
	data := c.Spec.Collector.Status.Config.Data[collectionagent.CollectorKindAgent.ConfigKey()]
	require.NoError(t, domain.UnmarshalYAML([]byte(data), &config))

	return config
}

func TestEnrichStatusAgent(t *testing.T) {
	config := moldedAgentConfig(t)

	pipelines := config["service"].(map[string]any)["pipelines"].(map[string]any)

	tests := []struct {
		name              string
		pipeline          string
		expectedReceivers []any
	}{
		{
			name:              "Metrics_UnionsDockerAndHost",
			pipeline:          "metrics",
			expectedReceivers: []any{"otlp/http", "otlp/grpc", "docker_stats", "hostmetrics"},
		},
		{
			name:              "Logs_UnionsFilelog",
			pipeline:          "logs",
			expectedReceivers: []any{"otlp/http", "otlp/grpc", "filelog"},
		},
		{
			name:              "Traces_BaseOnly",
			pipeline:          "traces",
			expectedReceivers: []any{"otlp/http", "otlp/grpc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := pipelines[tt.pipeline].(map[string]any)

			assert.Equal(t, tt.expectedReceivers, pipeline["receivers"])
		})
	}
}

func TestEnrichStatusAgentBodies(t *testing.T) {
	config := moldedAgentConfig(t)

	// The docker detector rides a full detectors list: bodies replace.
	resourcedetection := config["processors"].(map[string]any)["resourcedetection"].(map[string]any)
	assert.Equal(t, []any{"env", "system", "docker"}, resourcedetection["detectors"])
	assert.Equal(t, "2s", resourcedetection["timeout"])

	// Scraper keys must survive the merge: null-valued keys would be stripped
	// by the merge patch, so the template writes them as empty maps.
	hostmetrics := config["receivers"].(map[string]any)["hostmetrics"].(map[string]any)
	scrapers := hostmetrics["scrapers"].(map[string]any)
	for _, scraper := range []string{"cpu", "memory", "disk", "filesystem", "network", "load", "paging", "process", "processes"} {
		assert.Contains(t, scrapers, scraper)
	}

	// Modern docker engines reject the receiver's 1.25 default API version;
	// the value must be a quoted string or it parses as a float.
	dockerStats := config["receivers"].(map[string]any)["docker_stats"].(map[string]any)
	assert.Equal(t, "1.44", dockerStats["api_version"])

	// The agent stays stateless: filelog starts at the end instead of
	// checkpointing offsets into a state volume.
	filelog := config["receivers"].(map[string]any)["filelog"].(map[string]any)
	assert.Equal(t, "end", filelog["start_at"])
	assert.NotContains(t, filelog, "storage")
}

func TestEnrichStatusNonAgentNoConfig(t *testing.T) {
	c := collectionagent.Default()
	c.Spec.Collector.Kind = collectionagent.CollectorKind{}

	enricher := newDockerComposeMoldingEnricher()
	require.NoError(t, enricher.EnrichStatus(context.Background(), v1alpha1.MoldingKindCollector, c))

	assert.Empty(t, c.Spec.Collector.Status.Config.Data)
}
