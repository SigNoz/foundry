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

const validEndpoint = "https://ingest.us.signoz.cloud:443"

type pipeline struct {
	Receivers  []string `json:"receivers"`
	Processors []string `json:"processors"`
	Exporters  []string `json:"exporters"`
}

// newCasting builds a CollectionAgent casting for the given collector kind, with
// env and an optional enricher contribution already stamped at the config key.
func newCasting(t *testing.T, kind collectionagent.CollectorKind, env map[string]string, contribution string) *collectionagent.Casting {
	t.Helper()
	c := collectionagent.Default()
	c.Spec.Collector.Kind = kind
	c.Spec.Collector.Spec.Env = env
	if contribution != "" {
		c.Spec.Collector.Status.Config.Set(kind.ConfigKey(), []byte(contribution))
	}
	return c
}

// moldOutput molds c and returns the rendered agent config.
func moldOutput(t *testing.T, c *collectionagent.Casting) string {
	t.Helper()
	require.NoError(t, New(slog.Default()).MoldV1Alpha1(context.Background(), c))
	out := c.Spec.Collector.Status.Config.Data[collectionagent.CollectorKindAgent.ConfigKey()]
	require.NotEmpty(t, out, "molding must produce the collector config")
	return out
}

// agentPipelines molds c and returns the rendered config's service pipelines.
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

// TestMoldAgent composes the enricher's contribution onto the base config: base
// pipeline members are restored and the contribution's additions merged in per
// listMergeStrategy.
func TestMoldAgent(t *testing.T) {
	base := pipeline{
		Receivers:  []string{"otlp"},
		Processors: []string{"memory_limiter", "resourcedetection", "batch"},
		Exporters:  []string{"otlphttp"},
	}
	env := map[string]string{"SIGNOZ_INGESTION_ENDPOINT": validEndpoint}

	tests := []struct {
		name         string
		contribution string
		want         map[string]pipeline
	}{
		{
			name: "BaseOnly_NoContribution",
			want: map[string]pipeline{"traces": base, "metrics": base, "logs": base},
		},
		{
			name:         "ReceiverDelta_RestoresBaseAndUnions",
			contribution: "service:\n  pipelines:\n    metrics:\n      receivers: [docker_stats]\n",
			want: map[string]pipeline{
				"metrics": {Receivers: []string{"otlp", "docker_stats"}, Processors: base.Processors, Exporters: base.Exporters},
				"traces":  base,
			},
		},
		{
			name:         "ProcessorDelta_OrderedBeforeBatch",
			contribution: "service:\n  pipelines:\n    metrics:\n      processors: [attributes/env]\n",
			want: map[string]pipeline{
				"metrics": {Receivers: base.Receivers, Processors: []string{"memory_limiter", "resourcedetection", "attributes/env", "batch"}, Exporters: base.Exporters},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCasting(t, collectionagent.CollectorKindAgent, env, tt.contribution)
			got := agentPipelines(t, c)
			for name, want := range tt.want {
				assert.Equal(t, want, got[name], name)
			}
			assert.Equal(t, collectionagent.CollectorKindAgent.String(), c.Spec.Collector.Status.Env["OTEL_COLLECTOR_ROLE"])
		})
	}
}

// TestMoldAgentIngestionKey renders the ingestion-key header only when the key
// env is set.
func TestMoldAgentIngestionKey(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantHeader bool
	}{
		{
			name:       "KeySet_HeaderPresent",
			env:        map[string]string{"SIGNOZ_INGESTION_ENDPOINT": validEndpoint, "SIGNOZ_INGESTION_KEY": "test-key"},
			wantHeader: true,
		},
		{
			name:       "KeyUnset_HeaderAbsent",
			env:        map[string]string{"SIGNOZ_INGESTION_ENDPOINT": validEndpoint},
			wantHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := moldOutput(t, newCasting(t, collectionagent.CollectorKindAgent, tt.env, ""))
			assert.Contains(t, out, "${env:SIGNOZ_INGESTION_ENDPOINT}")
			if tt.wantHeader {
				assert.Contains(t, out, "signoz-ingestion-key")
			} else {
				assert.NotContains(t, out, "signoz-ingestion-key")
			}
		})
	}
}

// TestMoldAgentErrors covers the molding's input validation.
func TestMoldAgentErrors(t *testing.T) {
	validEnv := map[string]string{"SIGNOZ_INGESTION_ENDPOINT": validEndpoint}

	tests := []struct {
		name         string
		kind         collectionagent.CollectorKind
		env          map[string]string
		contribution string
		wantCode     int
	}{
		{
			name:     "UnknownKind",
			kind:     collectionagent.CollectorKind{},
			env:      validEnv,
			wantCode: foundryerrors.TypeUnsupported.ExitCode(),
		},
		{
			name:     "MissingIngestionEndpoint",
			kind:     collectionagent.CollectorKindAgent,
			env:      map[string]string{},
			wantCode: foundryerrors.TypeInvalidInput.ExitCode(),
		},
		{
			name:         "InvalidContributionYAML",
			kind:         collectionagent.CollectorKindAgent,
			env:          validEnv,
			contribution: "- not\n- a\n- map\n",
			wantCode:     foundryerrors.TypeInvalidInput.ExitCode(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCasting(t, tt.kind, tt.env, tt.contribution)
			err := New(slog.Default()).MoldV1Alpha1(context.Background(), c)
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, foundryerrors.ExitCode(err))
		})
	}
}
