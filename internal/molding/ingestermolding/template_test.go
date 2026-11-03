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
	assert.NotEmpty(t, ConfigV014415Template)
	assert.NotEmpty(t, OpampV0129xTemplate)
}

func TestConfigTemplate(t *testing.T) {
	tests := []struct {
		name             string
		image            string
		expectedTemplate *domain.Template
	}{
		{"LatestTag_NewestConfig", "signoz/signoz-otel-collector:latest", ConfigV014415Template},
		{"CommitTag_NewestConfig", "signoz/signoz-otel-collector:main-0efa06f", ConfigV014415Template},
		{"BelowFloor_OldConfig", "signoz/signoz-otel-collector:v0.144.5", ConfigV0129xTemplate},
		{"AtFloor_NewConfig", "signoz/signoz-otel-collector:0.144.6", ConfigV01446Template},
		{"AboveFloor_NewConfig", "signoz/signoz-otel-collector:v0.144.11", ConfigV01446Template},
		{"BelowNormalizerFloor_NewConfig", "signoz/signoz-otel-collector:v0.144.14", ConfigV01446Template},
		{"AtNormalizerFloor_NewestConfig", "signoz/signoz-otel-collector:0.144.15", ConfigV014415Template},
		{"AboveNormalizerFloor_NewestConfig", "signoz/signoz-otel-collector:v0.144.16", ConfigV014415Template},
		{"OldRelease_OldConfig", "signoz/signoz-otel-collector:v0.129.0", ConfigV0129xTemplate},
		{"UnstatedImage_NewestConfig", "", ConfigV014415Template},
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

	tests := []struct {
		name                     string
		template                 *domain.Template
		expectedAIProcessorsHeld bool
		expectedStaleFilterHeld  bool
		expectedNormalizerHeld   bool
		pass                     bool
	}{
		{"NewestConfig_HoldsAllProcessors", ConfigV014415Template, true, true, true, true},
		{"NewConfig_HoldsNoNormalizer", ConfigV01446Template, true, true, false, true},
		{"OldConfig_HoldsNoProcessors", ConfigV0129xTemplate, false, false, false, true},
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

			pipeline, err := material.GetStringSlice("service.pipelines.traces.processors")
			require.NoError(t, err)

			for _, processor := range aiProcessors {
				_, err := material.GetBytes("processors." + processor)
				if tt.expectedAIProcessorsHeld {
					assert.NoError(t, err, "%s is not configured", processor)
					assert.Contains(t, pipeline, processor)

					continue
				}

				assert.Error(t, err, "%s is configured", processor)
				assert.NotContains(t, pipeline, processor)
			}

			metricsPipeline, err := material.GetStringSlice("service.pipelines.metrics.processors")
			require.NoError(t, err)

			logsPipeline, err := material.GetStringSlice("service.pipelines.logs.processors")
			require.NoError(t, err)

			_, err = material.GetBytes("processors.signozlogsnormalizer")
			if tt.expectedNormalizerHeld {
				assert.NoError(t, err, "signozlogsnormalizer is not configured")
				assert.Equal(t, []string{"signozlogsnormalizer", "filter/drop_stale", "batch"}, logsPipeline)
			} else {
				assert.Error(t, err, "signozlogsnormalizer is configured")
				assert.NotContains(t, logsPipeline, "signozlogsnormalizer")
			}

			pipelines := [][]string{pipeline, metricsPipeline, logsPipeline}

			_, err = material.GetBytes("processors.filter/drop_stale")
			if !tt.expectedStaleFilterHeld {
				assert.Error(t, err, "filter/drop_stale is configured")

				for _, processors := range pipelines {
					assert.NotContains(t, processors, "filter/drop_stale")
				}

				return
			}

			assert.NoError(t, err, "filter/drop_stale is not configured")

			for _, processors := range pipelines {
				require.GreaterOrEqual(t, len(processors), 2)
				assert.Equal(t, []string{"filter/drop_stale", "batch"}, processors[len(processors)-2:])
			}
		})
	}
}
