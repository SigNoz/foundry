package gcpkuberneteskustomizecasting

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	collectionagentcasting "github.com/signoz/foundry/internal/casting/collectionagent"
	"github.com/signoz/foundry/internal/casting/collectionagent/kuberneteskustomizecasting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The molded detector list, and per detector the attributes it turns off.
func moldedDetection(t *testing.T, casting collectionagentcasting.Casting, kind collectionagent.CollectorKind) ([]string, map[string][]string, error) {
	t.Helper()

	ctx := context.Background()

	config := collectionagent.Default()
	config.Spec.Collector.Kind = kind

	enricher, err := casting.Enricher(ctx, config)
	require.NoError(t, err)

	if err := enricher.EnrichStatus(ctx, v1alpha1.MoldingKindCollector, config); err != nil {
		return nil, nil, err
	}

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

	return detectors, disabled, nil
}

func TestTemplatesRender(t *testing.T) {
	for name, test := range map[string]struct {
		template *domain.Template
		pass     bool
	}{
		"Agent_Valid":      {agentYAMLTemplate, true},
		"Deployment_Valid": {deploymentYAMLTemplate, true},
	} {
		t.Run(name, func(t *testing.T) {
			material, err := test.template.Render(nil, "collector.yaml")

			if test.pass {
				require.NoError(t, err)
				assert.NotEmpty(t, material.FmtContents())

				return
			}

			require.Error(t, err)
		})
	}
}

func TestMoldedResourceDetection(t *testing.T) {
	for _, test := range []struct {
		name              string
		kind              collectionagent.CollectorKind
		pass              bool
		expectedErrorType int
		expectedDetectors []string
		expectedDisabled  map[string][]string
	}{
		{
			"Agent_Valid", collectionagent.CollectorKindAgent, true, 0,
			[]string{"k8snode", "env", "gcp", "system"},
			map[string][]string{"gcp": {"host.id", "host.name", "k8s.cluster.name"}, "system": {"host.id", "host.name"}},
		},
		{
			"Deployment_Valid", collectionagent.CollectorKindDeployment, true, 0,
			[]string{"env", "gcp"},
			map[string][]string{"gcp": {"gcp.gce.instance.hostname", "gcp.gce.instance.name", "host.id", "host.name", "host.type", "k8s.cluster.name"}},
		},
		{"Sidecar_Invalid", collectionagent.CollectorKindSidecar, false, foundryerrors.TypeUnsupported.ExitCode(), nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			detectors, disabled, err := moldedDetection(t, New(slog.New(slog.DiscardHandler)), test.kind)

			if !test.pass {
				require.Error(t, err)
				assert.Equal(t, test.expectedErrorType, foundryerrors.ExitCode(err))

				return
			}

			require.NoError(t, err)

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
		pass bool
	}{
		{"Agent_Valid", collectionagent.CollectorKindAgent, true},
		{"Deployment_Valid", collectionagent.CollectorKindDeployment, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)

			generic, _, err := moldedDetection(t, kuberneteskustomizecasting.New(logger), test.kind)
			require.NoError(t, err)

			platform, _, err := moldedDetection(t, New(logger), test.kind)

			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			kept := slices.DeleteFunc(platform, func(detector string) bool { return !slices.Contains(generic, detector) })
			assert.Equal(t, generic, kept)
		})
	}
}
