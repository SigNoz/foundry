package ecsfargateterraformcasting

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sidecarCasting(t *testing.T) *collectionagent.Casting {
	t.Helper()
	config := collectionagent.Default()
	config.Spec.Collector.Kind = collectionagent.CollectorKindSidecar

	return config
}

func TestTemplatesRender(t *testing.T) {
	for name, tmpl := range map[string]*domain.Template{
		"Versions_Valid": versionsTF,
		"Main_Valid":     mainTF,
		"Outputs_Valid":  outputsTF,
		"Sidecar_Valid":  sidecarYAMLTemplate,
	} {
		t.Run(name, func(t *testing.T) {
			material, err := tmpl.Render(templateDataFor(*sidecarCasting(t)), strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
			require.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}

// A fargate task has no container instance, and a sidecar rides in every task.
func TestRefusals(t *testing.T) {
	for _, test := range []struct {
		name     string
		kind     collectionagent.CollectorKind
		replicas *int
		pass     bool
	}{
		{"Sidecar_Valid", collectionagent.CollectorKindSidecar, nil, true},
		{"Agent_Invalid", collectionagent.CollectorKindAgent, nil, false},
		{"Replicas_Invalid", collectionagent.CollectorKindSidecar, domain.NewIntPtr(2), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := sidecarCasting(t)
			config.Spec.Collector.Kind = test.kind
			config.Spec.Collector.Spec.Cluster.Replicas = test.replicas

			err := newEcsFargateMoldingEnricher().EnrichStatus(context.Background(), v1alpha1.MoldingKindCollector, config)
			if err == nil {
				err = New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *config, pourer.New("collectionagent"))
			}

			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

// What ECS adds rides in the enricher and has to survive the molding's merge.
func TestSidecarConfig(t *testing.T) {
	config := sidecarCasting(t)
	require.NoError(t, newEcsFargateMoldingEnricher().EnrichStatus(context.Background(), v1alpha1.MoldingKindCollector, config))
	require.NoError(t, collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(context.Background(), config))

	merged := domain.MustNewYAMLMaterial([]byte(config.Spec.Collector.Status.Config.Data[collectionagent.CollectorKindSidecar.ConfigKey()]), "sidecar.yaml")

	endpoint, err := merged.GetBytes("receivers.fluentforward.endpoint")
	require.NoError(t, err)
	assert.Equal(t, "unix:///var/run/fluent.sock", string(endpoint))

	for path, expected := range map[string][]string{
		"processors.resourcedetection.detectors": {"env", "ecs"},
		"service.pipelines.logs.receivers":       {"otlp/http", "otlp/grpc", "fluentforward"},
		"service.pipelines.logs.processors":      {"memory_limiter", "resourcedetection", "transform/firelens", "batch"},
		"service.pipelines.metrics.processors":   {"memory_limiter", "resourcedetection", "filter/ecs", "batch"},
	} {
		actual, err := merged.GetStringSlice(path)
		require.NoError(t, err)
		assert.Equal(t, expected, actual, path)
	}

	statements, err := merged.GetStringSlice("processors.transform/firelens.log_statements")
	require.NoError(t, err)
	assert.Len(t, statements, 2)
}
