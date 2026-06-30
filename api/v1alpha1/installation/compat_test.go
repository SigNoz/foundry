package installation

import (
	"io"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestCheckCompatibility(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name       string
		collector  string
		clickhouse string
		pass       bool
	}{
		{"NewCollector_OldClickhouse_Fails", "signoz/signoz-otel-collector:0.144.6", "clickhouse/clickhouse-server:25.5.6", false},
		{"NewCollector_NewClickhouse_OK", "signoz/signoz-otel-collector:0.144.6", "clickhouse/clickhouse-server:25.12.5", true},
		{"NewCollector_NewClickhouseAlpine_OK", "signoz/signoz-otel-collector:0.144.6", "clickhouse/clickhouse-server:25.12.5-alpine", true},
		{"FloorCollector_OldClickhouse_OK", "signoz/signoz-otel-collector:0.144.5", "clickhouse/clickhouse-server:25.5.6", true},
		{"LatestCollector_OldClickhouse_WarnsNoError", "signoz/signoz-otel-collector:latest", "clickhouse/clickhouse-server:25.5.6", true},
		{"LatestCollector_NewClickhouse_OK", "signoz/signoz-otel-collector:latest", "clickhouse/clickhouse-server:25.12.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := &Casting{}
			casting.Spec.Ingester.Spec.Enabled = v1alpha1.BoolPtr(true)
			casting.Spec.Ingester.Spec.Image = tt.collector
			casting.Spec.TelemetryStore.Spec.Enabled = v1alpha1.BoolPtr(true)
			casting.Spec.TelemetryStore.Spec.Image = tt.clickhouse

			err := casting.CheckCompatibility(logger)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCheckCompatibilityDisabledComponentSkips(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	casting := &Casting{}
	casting.Spec.Ingester.Spec.Enabled = v1alpha1.BoolPtr(false)
	casting.Spec.Ingester.Spec.Image = "signoz/signoz-otel-collector:0.144.6"
	casting.Spec.TelemetryStore.Spec.Enabled = v1alpha1.BoolPtr(true)
	casting.Spec.TelemetryStore.Spec.Image = "clickhouse/clickhouse-server:25.5.6"

	assert.NoError(t, casting.CheckCompatibility(logger))
}
