package azurekuberneteshelmcasting

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/casting/collectionagent/kuberneteshelmcasting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The molded detector list, and per detector the attributes it turns off.
func moldedDetection(t *testing.T, casting helmCasting, kind collectionagent.CollectorKind) ([]string, map[string][]string) {
	t.Helper()

	ctx := context.Background()

	config := collectionagent.Default()
	config.Spec.Collector.Kind = kind

	enricher, err := casting.Enricher(ctx, config)
	require.NoError(t, err)

	require.NoError(t, enricher.EnrichStatus(ctx, v1alpha1.MoldingKindCollector, config))
	require.NoError(t, collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(ctx, config))
	require.NoError(t, config.MergeStatusIntoSpec())

	var molded struct {
		Processors struct {
			ResourceDetection map[string]json.RawMessage `json:"resourcedetection"`
		} `json:"processors"`
	}
	require.NoError(t, domain.UnmarshalYAML([]byte(config.Spec.Collector.Spec.Config.Data[kind.ConfigKey()]), &molded))

	detection := molded.Processors.ResourceDetection

	var detectors []string
	require.NoError(t, json.Unmarshal(detection["detectors"], &detectors))

	disabled := map[string][]string{}

	for _, name := range detectors {
		raw, stated := detection[name]
		if !stated {
			continue
		}

		var detector struct {
			ResourceAttributes map[string]struct {
				Enabled bool `json:"enabled"`
			} `json:"resource_attributes"`
		}
		require.NoError(t, json.Unmarshal(raw, &detector))

		for attribute, setting := range detector.ResourceAttributes {
			if !setting.Enabled {
				disabled[name] = append(disabled[name], attribute)
			}
		}

		slices.Sort(disabled[name])
	}

	return detectors, disabled
}

func TestTemplatesRender(t *testing.T) {
	for name, tmpl := range map[string]*domain.Template{
		"Agent_Valid":      agentYAMLTemplate,
		"Deployment_Valid": deploymentYAMLTemplate,
	} {
		t.Run(name, func(t *testing.T) {
			material, err := tmpl.Render(nil, "collector.yaml")
			require.NoError(t, err)

			assert.NotEmpty(t, material.FmtContents())
		})
	}
}

func TestMoldedResourceDetection(t *testing.T) {
	for _, test := range []struct {
		name              string
		kind              collectionagent.CollectorKind
		expectedDetectors []string
		expectedDisabled  map[string][]string
	}{
		{
			"Agent_Valid", collectionagent.CollectorKindAgent,
			[]string{"k8snode", "env", "aks", "azure", "system"},
			map[string][]string{"azure": {"host.id", "host.name"}, "system": {"host.id", "host.name"}},
		},
		{
			"Deployment_Valid", collectionagent.CollectorKindDeployment,
			[]string{"env", "aks", "azure"},
			map[string][]string{"azure": {"azure.resourcegroup.name", "azure.vm.name", "azure.vm.scaleset.name", "azure.vm.size", "host.id", "host.name"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			detectors, disabled := moldedDetection(t, New(slog.New(slog.DiscardHandler)), test.kind)

			assert.Equal(t, test.expectedDetectors, detectors)
			assert.Equal(t, test.expectedDisabled, disabled)
		})
	}
}

// The delta replaces the generic detector list, so it must keep every generic detector in order.
func TestMoldedDetectorsKeepGeneric(t *testing.T) {
	for _, test := range []struct {
		name string
		kind collectionagent.CollectorKind
	}{
		{"Agent_Valid", collectionagent.CollectorKindAgent},
		{"Deployment_Valid", collectionagent.CollectorKindDeployment},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)

			generic, _ := moldedDetection(t, kuberneteshelmcasting.New(logger), test.kind)
			platform, _ := moldedDetection(t, New(logger), test.kind)

			kept := slices.DeleteFunc(platform, func(detector string) bool { return !slices.Contains(generic, detector) })
			assert.Equal(t, generic, kept)
		})
	}
}
