package mechanic

import (
	"context"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalTables(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected []string
		pass     bool
	}{
		{
			name:     "RootCompositeQuery_Traces",
			data:     `{"compositeQuery":{"queries":[{"type":"builder_query","spec":{"signal":"traces"}}]}}`,
			expected: []string{"signoz_traces.distributed_signoz_index_v3"},
			pass:     true,
		},
		{
			name:     "ConditionCompositeQuery_Logs",
			data:     `{"condition":{"compositeQuery":{"queries":[{"spec":{"signal":"logs"}}]}}}`,
			expected: []string{"signoz_logs.distributed_logs_v2"},
			pass:     true,
		},
		{
			name:     "Metrics",
			data:     `{"compositeQuery":{"queries":[{"spec":{"signal":"metrics"}}]}}`,
			expected: []string{"signoz_metrics.distributed_samples_v4"},
			pass:     true,
		},
		{
			name:     "MultipleSignalsDeduped",
			data:     `{"compositeQuery":{"queries":[{"spec":{"signal":"traces"}},{"spec":{"signal":"logs"}},{"spec":{"signal":"traces"}}]}}`,
			expected: []string{"signoz_traces.distributed_signoz_index_v3", "signoz_logs.distributed_logs_v2"},
			pass:     true,
		},
		{
			name:     "NoCompositeQuery",
			data:     `{"alert":"some name"}`,
			expected: nil,
			pass:     true,
		},
		{
			name: "UnknownSignal",
			data: `{"compositeQuery":{"queries":[{"spec":{"signal":"events"}}]}}`,
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables, err := SignalTables([]byte(tt.data))
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, tables)
		})
	}
}

func TestTelemetryStoreQuery(t *testing.T) {
	c := installation.Default()
	c.Spec.Deployment = v1alpha1.TypeDeployment{Mode: v1alpha1.ModeDocker, Flavor: v1alpha1.FlavorCompose}
	c.Spec.TelemetryStore.Status.Addresses.TCP = []string{"tcp://dev-telemetrystore-clickhouse-0-0:9000"}

	exec := &fakeExecutor{out: []byte("25.5.6.1\n")}

	store, err := NewTelemetryStore(exec, c)
	require.NoError(t, err)

	out, err := store.Query(context.Background(), "SELECT version()")
	require.NoError(t, err)

	assert.Equal(t, "25.5.6.1", out)
	require.Len(t, exec.calls, 1)
	assert.Equal(t, "docker", exec.calls[0].name)
	assert.Equal(t, []string{
		"exec", "dev-telemetrystore-clickhouse-0-0",
		"clickhouse-client", "--query", "SELECT version()",
	}, exec.calls[0].args)
}

func TestNewTelemetryStoreUnsupported(t *testing.T) {
	c := installation.Default()
	// default deployment is not docker/compose
	_, err := NewTelemetryStore(&fakeExecutor{}, c)
	assert.Error(t, err)
}

func TestNewTelemetryStoreMissingAddress(t *testing.T) {
	c := installation.Default()
	c.Spec.Deployment = v1alpha1.TypeDeployment{Mode: v1alpha1.ModeDocker, Flavor: v1alpha1.FlavorCompose}
	c.Spec.TelemetryStore.Status.Addresses.TCP = nil

	_, err := NewTelemetryStore(&fakeExecutor{}, c)
	assert.Error(t, err)
}

func TestTelemetryStoreQueryError(t *testing.T) {
	c := installation.Default()
	c.Spec.Deployment = v1alpha1.TypeDeployment{Mode: v1alpha1.ModeDocker, Flavor: v1alpha1.FlavorCompose}
	c.Spec.TelemetryStore.Status.Addresses.TCP = []string{"tcp://dev-telemetrystore-clickhouse-0-0:9000"}

	store, err := NewTelemetryStore(&fakeExecutor{err: assert.AnError}, c)
	require.NoError(t, err)

	_, err = store.Query(context.Background(), "SELECT version()")
	assert.Error(t, err)
}
