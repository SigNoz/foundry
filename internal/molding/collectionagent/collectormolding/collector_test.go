package collectormolding

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectorTemplateLoaded(t *testing.T) {
	t.Parallel()
	assert.NotEmpty(t, AgentYAMLTemplate)
}

func newCasting(kind collectionagent.CollectorKind) *collectionagent.Casting {
	c := collectionagent.Default()
	c.Spec.Collector.Kind = kind
	c.Spec.Collector.Spec.Env = map[string]string{
		"SIGNOZ_INGESTION_ENDPOINT": "https://ingest.us.signoz.cloud:443",
		"SIGNOZ_INGESTION_KEY":      "test-key",
	}
	return c
}

func TestMoldAgent_RendersExpectedConfig(t *testing.T) {
	t.Parallel()

	c := newCasting(collectionagent.CollectorKindAgent)
	c.Spec.Collector.Status.Receivers = map[string]collectionagent.Component{
		"hostmetrics": {
			Body:      map[string]any{"collection_interval": "60s"},
			Pipelines: []string{"metrics"},
		},
		"filelog": {
			Body:      map[string]any{"include": []any{"/var/log/*.log"}},
			Pipelines: []string{"logs"},
		},
	}
	c.Spec.Collector.Status.ResourceDetectors = []string{"docker"}

	m := New(slog.Default())
	err := m.MoldV1Alpha1(context.Background(), c)
	require.NoError(t, err)

	role := collectionagent.CollectorKindAgent.String()
	expectedPath := filepath.Join(v1alpha1.MoldingKindCollector.String(), role, role+".yaml")
	out, ok := c.Spec.Collector.Status.Config.Data[expectedPath]
	require.True(t, ok, "molding must stamp %s", expectedPath)
	assert.Contains(t, out, "otlp:")
	assert.Contains(t, out, "memory_limiter:")
	assert.Contains(t, out, "resourcedetection:")
	assert.Contains(t, out, "batch:")
	assert.Contains(t, out, "otlphttp:")
	assert.Contains(t, out, "${env:SIGNOZ_INGESTION_ENDPOINT}")
	assert.Contains(t, out, "${env:SIGNOZ_INGESTION_KEY}")
	assert.Contains(t, out, "hostmetrics:")
	assert.Contains(t, out, "filelog:")
	assert.Contains(t, out, "- docker")
	assert.Contains(t, out, "traces:")
	assert.Contains(t, out, "metrics:")
	assert.Contains(t, out, "logs:")
	assert.Equal(t, role, c.Spec.Collector.Status.Env["OTEL_COLLECTOR_ROLE"])
}

func TestMoldAgent_RejectsUnknownPipelineName(t *testing.T) {
	t.Parallel()

	c := newCasting(collectionagent.CollectorKindAgent)
	c.Spec.Collector.Status.Receivers = map[string]collectionagent.Component{
		"hostmetrics": {
			Body:      map[string]any{"collection_interval": "60s"},
			Pipelines: []string{"events"}, // not one of traces|metrics|logs
		},
	}

	err := New(slog.Default()).MoldV1Alpha1(context.Background(), c)
	require.Error(t, err)
	assert.Equal(t, foundryerrors.TypeInvalidInput.ExitCode(), foundryerrors.ExitCode(err))
	assert.True(t, strings.Contains(err.Error(), "events"))
}

func TestMoldAgent_RejectsMissingIngestionEndpoint(t *testing.T) {
	t.Parallel()

	c := newCasting(collectionagent.CollectorKindAgent)
	c.Spec.Collector.Spec.Env = map[string]string{} // endpoint absent

	err := New(slog.Default()).MoldV1Alpha1(context.Background(), c)
	require.Error(t, err)
	assert.Equal(t, foundryerrors.TypeInvalidInput.ExitCode(), foundryerrors.ExitCode(err))
	assert.True(t, strings.Contains(err.Error(), "SIGNOZ_INGESTION_ENDPOINT"))
}

func TestMoldAgent_OmitsIngestionKeyHeaderWhenNotSet(t *testing.T) {
	t.Parallel()

	c := newCasting(collectionagent.CollectorKindAgent)
	// Self-hosted deployment: endpoint only, no key.
	c.Spec.Collector.Spec.Env = map[string]string{
		"SIGNOZ_INGESTION_ENDPOINT": "http://otel-collector.signoz-headless.svc:4318",
	}

	err := New(slog.Default()).MoldV1Alpha1(context.Background(), c)
	require.NoError(t, err)

	role := collectionagent.CollectorKindAgent.String()
	out := c.Spec.Collector.Status.Config.Data[filepath.Join(v1alpha1.MoldingKindCollector.String(), role, role+".yaml")]
	assert.Contains(t, out, "${env:SIGNOZ_INGESTION_ENDPOINT}")
	assert.NotContains(t, out, "signoz-ingestion-key")
	assert.NotContains(t, out, "${env:SIGNOZ_INGESTION_KEY}")
}

func TestMoldUnknownKind_ReturnsUnsupported(t *testing.T) {
	t.Parallel()

	c := collectionagent.Default()
	c.Spec.Collector.Kind = collectionagent.CollectorKind{} // zero-value

	err := New(slog.Default()).MoldV1Alpha1(context.Background(), c)
	require.Error(t, err)
	assert.Equal(t, foundryerrors.TypeUnsupported.ExitCode(), foundryerrors.ExitCode(err))
}

// Sanity that the molding still announces its kind.
func TestCollectorKind(t *testing.T) {
	t.Parallel()
	assert.Equal(t, v1alpha1.MoldingKindCollector, New(slog.Default()).Kind())
}
