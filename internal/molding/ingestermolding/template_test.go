package ingestermolding

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngester(t *testing.T) {
	assert.NotEmpty(t, ConfigV0129xTemplate)
	assert.NotEmpty(t, ConfigV01446Template)
	assert.NotEmpty(t, ConfigV014414Template)
	assert.NotEmpty(t, OpampV0129xTemplate)
}

func TestConfigTemplate(t *testing.T) {
	tests := []struct {
		name             string
		image            string
		expectedTemplate *domain.Template
	}{
		{"LatestTag_NewestConfig", "signoz/signoz-otel-collector:latest", ConfigV014414Template},
		{"CommitTag_NewestConfig", "signoz/signoz-otel-collector:main-0efa06f", ConfigV014414Template},
		{"BelowFloor_OldConfig", "signoz/signoz-otel-collector:v0.144.5", ConfigV0129xTemplate},
		{"AtFloor_NewConfig", "signoz/signoz-otel-collector:0.144.6", ConfigV01446Template},
		{"AboveFloor_NewConfig", "signoz/signoz-otel-collector:v0.144.11", ConfigV01446Template},
		{"BelowNormalizerFloor_NewConfig", "signoz/signoz-otel-collector:v0.144.13", ConfigV01446Template},
		{"AtNormalizerFloor_NewestConfig", "signoz/signoz-otel-collector:0.144.14", ConfigV014414Template},
		{"AboveNormalizerFloor_NewestConfig", "signoz/signoz-otel-collector:v0.144.15", ConfigV014414Template},
		{"OldRelease_OldConfig", "signoz/signoz-otel-collector:v0.129.0", ConfigV0129xTemplate},
		{"UnstatedImage_NewestConfig", "", ConfigV014414Template},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Same(t, tt.expectedTemplate, configTemplate(v1alpha1.MoldingSpec{Image: tt.image}))
		})
	}
}

func TestConfigTemplateRender(t *testing.T) {
	data := Data{
		SignozOpampAddress:            "http://signoz:4320",
		TelemetryStoreTracesAddress:   "tcp://clickhouse:9000/signoz_traces",
		TelemetryStoreMetricsAddress:  "tcp://clickhouse:9000/signoz_metrics",
		TelemetryStoreLogsAddress:     "tcp://clickhouse:9000/signoz_logs",
		TelemetryStoreMeterAddress:    "tcp://clickhouse:9000/signoz_meter",
		TelemetryStoreMetadataAddress: "tcp://clickhouse:9000/signoz_metadata",
	}

	aiProcessors := []string{"signozspanmapper", "signozllmpricing"}
	logsProcessors := []string{"signozlogsnormalizer", "filter/drop_stale"}

	tests := []struct {
		name                       string
		template                   *domain.Template
		expectedAIProcessorsHeld   bool
		expectedLogsProcessorsHeld bool
		pass                       bool
	}{
		{"NewestConfig_HoldsAllProcessors", ConfigV014414Template, true, true, true},
		{"NewConfig_HoldsAIProcessorsOnly", ConfigV01446Template, true, false, true},
		{"OldConfig_HoldsNoProcessors", ConfigV0129xTemplate, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			err := tt.template.Execute(buf, data)
			if !tt.pass {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)

			material, err := domain.NewYAMLMaterial(buf.Bytes(), "ingester.yaml")
			require.NoError(t, err)

			tracesPipeline, err := material.GetStringSlice("service.pipelines.traces.processors")
			require.NoError(t, err)

			metricsPipeline, err := material.GetStringSlice("service.pipelines.metrics.processors")
			require.NoError(t, err)

			logsPipeline, err := material.GetStringSlice("service.pipelines.logs.processors")
			require.NoError(t, err)

			assertProcessorsHeld(t, material, tracesPipeline, aiProcessors, tt.expectedAIProcessorsHeld)
			assertProcessorsHeld(t, material, logsPipeline, logsProcessors, tt.expectedLogsProcessorsHeld)

			if !tt.expectedLogsProcessorsHeld {
				return
			}

			assert.Equal(t, []string{"signozlogsnormalizer", "filter/drop_stale", "batch"}, logsPipeline)
			assert.Contains(t, metricsPipeline, "filter/drop_stale")
			assert.Contains(t, tracesPipeline, "filter/drop_stale")
		})
	}
}

func assertProcessorsHeld(t *testing.T, material domain.YAMLMaterial, pipeline []string, processors []string, expectedHeld bool) {
	t.Helper()

	for _, processor := range processors {
		_, err := material.GetBytes("processors." + processor)
		if expectedHeld {
			assert.NoError(t, err, "%s is not configured", processor)
			assert.Contains(t, pipeline, processor)

			continue
		}

		assert.Error(t, err, "%s is configured", processor)
		assert.NotContains(t, pipeline, processor)
	}
}
