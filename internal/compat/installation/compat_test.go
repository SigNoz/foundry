package installation

import (
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/stretchr/testify/assert"
)

func TestCheckCompatibility(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	tests := []struct {
		name              string
		signoz            string
		collector         string
		clickhouse        string
		collectorDisabled bool
		pass              bool
	}{
		{"NewCollector_OldClickhouse_Fails", "signoz/signoz:latest", "signoz/signoz-otel-collector:0.144.11", "clickhouse/clickhouse-server:25.5.6", false, false},
		{"NewCollector_NewClickhouse_OK", "signoz/signoz:latest", "signoz/signoz-otel-collector:0.144.11", "clickhouse/clickhouse-server:25.12.5", false, true},
		{"NewCollector_NewClickhouseAlpine_OK", "signoz/signoz:latest", "signoz/signoz-otel-collector:0.144.11", "clickhouse/clickhouse-server:25.12.5-alpine", false, true},
		{"FloorCollector_OldClickhouse_OK", "signoz/signoz:latest", "signoz/signoz-otel-collector:0.144.5", "clickhouse/clickhouse-server:25.5.6", false, true},
		{"LatestCollector_OldClickhouse_WarnsNoError", "signoz/signoz:latest", "signoz/signoz-otel-collector:latest", "clickhouse/clickhouse-server:25.5.6", false, true},
		{"LatestCollector_NewClickhouse_OK", "signoz/signoz:latest", "signoz/signoz-otel-collector:latest", "clickhouse/clickhouse-server:25.12.5", false, true},
		{"DisabledCollector_OldClickhouse_OK", "signoz/signoz:latest", "signoz/signoz-otel-collector:0.144.11", "clickhouse/clickhouse-server:25.5.6", true, true},
		{"NewSignoz_OldCollector_Fails", "signoz/signoz:0.143.0", "signoz/signoz-otel-collector:0.144.5", "clickhouse/clickhouse-server:25.12.5", false, false},
		{"NewSignoz_FloorCollector_OK", "signoz/signoz:0.143.0", "signoz/signoz-otel-collector:0.144.6", "clickhouse/clickhouse-server:25.12.5", false, true},
		{"NewSignoz_NewCollector_OK", "signoz/signoz:0.143.0", "signoz/signoz-otel-collector:0.144.11", "clickhouse/clickhouse-server:25.12.5", false, true},
		{"LatestSignoz_OldCollector_WarnsNoError", "signoz/signoz:latest", "signoz/signoz-otel-collector:0.144.5", "clickhouse/clickhouse-server:25.12.5", false, true},
		{"OldSignoz_OldCollector_OK", "signoz/signoz:0.142.1", "signoz/signoz-otel-collector:0.144.5", "clickhouse/clickhouse-server:25.5.6", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := &installation.Casting{}
			casting.Spec.Ingester.Spec.Enabled = v1alpha1.BoolPtr(!tt.collectorDisabled)
			casting.Spec.Ingester.Spec.Image = tt.collector
			casting.Spec.TelemetryStore.Spec.Enabled = v1alpha1.BoolPtr(true)
			casting.Spec.TelemetryStore.Spec.Image = tt.clickhouse
			casting.Spec.Signoz.Spec.Enabled = v1alpha1.BoolPtr(true)
			casting.Spec.Signoz.Spec.Image = tt.signoz

			err := Compatibility(casting, logger)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
