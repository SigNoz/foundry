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
	assert.NotEmpty(t, ConfigV014411Template)
	assert.NotEmpty(t, OpampV0129xTemplate)
}

func TestConfigTemplate(t *testing.T) {
	tests := []struct {
		name             string
		image            string
		expectedTemplate *domain.Template
	}{
		{"LatestTag_NewConfig", "signoz/signoz-otel-collector:latest", ConfigV014411Template},
		{"CommitTag_NewConfig", "signoz/signoz-otel-collector:main-0efa06f", ConfigV014411Template},
		{"BelowFloor_OldConfig", "signoz/signoz-otel-collector:v0.144.10", ConfigV0129xTemplate},
		{"AtFloor_NewConfig", "signoz/signoz-otel-collector:0.144.11", ConfigV014411Template},
		{"AboveFloor_NewConfig", "signoz/signoz-otel-collector:v0.144.12", ConfigV014411Template},
		{"OldRelease_OldConfig", "signoz/signoz-otel-collector:v0.129.0", ConfigV0129xTemplate},
		{"UnstatedImage_NewConfig", "", ConfigV014411Template},
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
	}{
		{"NewConfig_HoldsAIProcessors", ConfigV014411Template, true},
		{"OldConfig_HoldsNoAIProcessors", ConfigV0129xTemplate, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			require.NoError(t, tt.template.Execute(buf, data))

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
		})
	}
}
