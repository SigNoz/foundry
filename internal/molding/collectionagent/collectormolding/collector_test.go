package collectormolding

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pipeline struct {
	Receivers  []string `json:"receivers"`
	Processors []string `json:"processors"`
	Exporters  []string `json:"exporters"`
}

// newCasting stamps the optional enricher config at the kind's config key.
func newCasting(t *testing.T, kind collectionagent.CollectorKind, enricherConfig string) *collectionagent.Casting {
	t.Helper()
	c := collectionagent.Default()
	c.Spec.Collector.Kind = kind
	if enricherConfig != "" {
		c.Spec.Collector.Status.Config.Set(kind.ConfigKey(), []byte(enricherConfig))
	}
	return c
}

func moldOutput(t *testing.T, c *collectionagent.Casting) string {
	t.Helper()
	require.NoError(t, New(slog.Default()).MoldV1Alpha1(context.Background(), c))
	out := c.Spec.Collector.Status.Config.Data[collectionagent.CollectorKindAgent.ConfigKey()]
	require.NotEmpty(t, out, "molding must produce the collector config")
	return out
}

// sigs.k8s.io/yaml routes through JSON, so json tags.
func agentPipelines(t *testing.T, c *collectionagent.Casting) map[string]pipeline {
	t.Helper()
	var cfg struct {
		Service struct {
			Pipelines map[string]pipeline `json:"pipelines"`
		} `json:"service"`
	}
	require.NoError(t, domain.UnmarshalYAML([]byte(moldOutput(t, c)), &cfg))
	return cfg.Service.Pipelines
}

// Base pipeline members are restored and the enricher config's additions merge
// in per listTypes.
func TestMoldAgent(t *testing.T) {
	base := pipeline{
		Receivers:  []string{"otlp/http", "otlp/grpc"},
		Processors: []string{"memory_limiter", "resourcedetection", "batch"},
		Exporters:  []string{"otlphttp/signoz"},
	}

	tests := []struct {
		name           string
		enricherConfig string
		expected       map[string]pipeline
	}{
		{
			name:     "BaseOnly_NoEnricherConfig",
			expected: map[string]pipeline{"traces": base, "metrics": base, "logs": base},
		},
		{
			name:           "ReceiverDelta_RestoresBaseAndUnions",
			enricherConfig: "service:\n  pipelines:\n    metrics:\n      receivers: [docker_stats]\n",
			expected: map[string]pipeline{
				"metrics": {Receivers: []string{"otlp/http", "otlp/grpc", "docker_stats"}, Processors: base.Processors, Exporters: base.Exporters},
				"traces":  base,
			},
		},
		{
			name:           "ProcessorDelta_OrderedBeforeBatch",
			enricherConfig: "service:\n  pipelines:\n    metrics:\n      processors: [attributes/env]\n",
			expected: map[string]pipeline{
				"metrics": {Receivers: base.Receivers, Processors: []string{"memory_limiter", "resourcedetection", "attributes/env", "batch"}, Exporters: base.Exporters},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCasting(t, collectionagent.CollectorKindAgent, tt.enricherConfig)
			got := agentPipelines(t, c)
			for name, expected := range tt.expected {
				assert.Equal(t, expected, got[name], name)
			}
			assert.Equal(t, collectionagent.CollectorKindAgent.String(), c.Spec.Collector.Status.Env["OTEL_COLLECTOR_ROLE"])
		})
	}
}

// Every kind molds its base config at its own config key and reports its
// role through the env.
func TestMoldKinds(t *testing.T) {
	for _, kind := range collectionagent.CollectorKinds() {
		t.Run(kind.String()+"_MoldsConfigAtKindKey", func(t *testing.T) {
			c := newCasting(t, kind, "")
			require.NoError(t, New(slog.Default()).MoldV1Alpha1(context.Background(), c))

			assert.NotEmpty(t, c.Spec.Collector.Status.Config.Data[kind.ConfigKey()])
			assert.Equal(t, kind.String(), c.Spec.Collector.Status.Env["OTEL_COLLECTOR_ROLE"])
		})
	}
}

func TestMoldAgentErrors(t *testing.T) {
	tests := []struct {
		name           string
		kind           collectionagent.CollectorKind
		enricherConfig string
		expectedCode   int
	}{
		{
			name:         "UnknownKind",
			kind:         collectionagent.CollectorKind{},
			expectedCode: foundryerrors.TypeUnsupported.ExitCode(),
		},
		{
			name:           "InvalidEnricherConfigYAML",
			kind:           collectionagent.CollectorKindAgent,
			enricherConfig: "- not\n- a\n- map\n",
			expectedCode:   foundryerrors.TypeInvalidInput.ExitCode(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCasting(t, tt.kind, tt.enricherConfig)
			err := New(slog.Default()).MoldV1Alpha1(context.Background(), c)
			require.Error(t, err)
			assert.Equal(t, tt.expectedCode, foundryerrors.ExitCode(err))
		})
	}
}
