package kuberneteshelmcasting

import (
	"context"
	"log/slog"
	"maps"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForge(t *testing.T) {
	for _, test := range []struct {
		name              string
		kind              collectionagent.CollectorKind
		controller        string
		pass              bool
		expectedErrorType int
		expectedPath      string
		expectedKubeStack bool
	}{
		{"Agent_Valid", collectionagent.CollectorKindAgent, "", true, 0, "collectionagent/collector/agent/values.yaml", false},
		{"Deployment_Valid", collectionagent.CollectorKindDeployment, "", true, 0, "collectionagent/collector/deployment/values.yaml", false},
		{"OperatorAgent_Valid", collectionagent.CollectorKindAgent, collectionagent.CollectorControllerOpenTelemetryOperator, true, 0, "collectionagent/collector/agent/values.yaml", true},
		{"OperatorDeployment_Valid", collectionagent.CollectorKindDeployment, collectionagent.CollectorControllerOpenTelemetryOperator, true, 0, "collectionagent/collector/deployment/values.yaml", true},
		{"UnknownController_Invalid", collectionagent.CollectorKindAgent, "argo", false, foundryerrors.TypeInvalidInput.ExitCode(), "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := moldedCasting(t, test.kind, defaultEnv())

			if test.controller != "" {
				config.Metadata.Annotations = map[string]string{collectionagent.KubernetesCollectorController.Key: test.controller}
			}

			p := pourer.New("collectionagent")
			err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *config, p)

			if test.pass {
				require.NoError(t, err)

				materials, err := p.Pour()
				require.NoError(t, err)
				require.Len(t, materials, 1)

				assert.Equal(t, test.expectedPath, materials[0].Path())

				var rendered map[string]any
				require.NoError(t, domain.UnmarshalYAML(materials[0].FmtContents(), &rendered))

				assert.Equal(t, test.expectedKubeStack, rendered["collectors"] != nil)
				assert.Equal(t, !test.expectedKubeStack, rendered["otelAgent"] != nil)

				return
			}

			require.Error(t, err)
			assert.Equal(t, test.expectedErrorType, foundryerrors.ExitCode(err))
		})
	}
}

func TestChartSource(t *testing.T) {
	casting := New(slog.New(slog.DiscardHandler))
	operator := collectionagent.CollectorControllerOpenTelemetryOperator

	for _, test := range []struct {
		name            string
		controller      string
		annotations     map[string]string
		pass            bool
		expectedChart   string
		expectedVersion string
		expectedRepoURL string
	}{
		{"Unstated_Valid", "", nil, true, "k8s-infra", "", "https://charts.signoz.io"},
		{
			"StatedVersion_Valid", "",
			map[string]string{collectionagent.HelmChartVersion.Key: "0.17.1"},
			true, "k8s-infra", "0.17.1", "https://charts.signoz.io",
		},
		{
			"SlashedRef_Valid", "",
			map[string]string{collectionagent.HelmChart.Key: "./charts/k8s-infra"},
			true, "./charts/k8s-infra", "", "",
		},
		{"OperatorUnstated_Valid", operator, nil, true, kubeStackChart, kubeStackChartVersion, kubeStackRepoURL},
		{
			"OperatorStatedLatest_Valid", operator,
			map[string]string{collectionagent.HelmChartVersion.Key: "latest"},
			true, kubeStackChart, "", kubeStackRepoURL,
		},
		{
			"OperatorStatedChartName_Valid", operator,
			map[string]string{collectionagent.HelmChart.Key: "k8s-infra"},
			true, "k8s-infra", "", kubeStackRepoURL,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Metadata.Annotations = map[string]string{}
			maps.Copy(config.Metadata.Annotations, test.annotations)

			if test.controller != "" {
				config.Metadata.Annotations[collectionagent.KubernetesCollectorController.Key] = test.controller
			}

			chart, version, repoURL := casting.chartSource(*config)

			assert.Equal(t, test.expectedChart, chart)
			assert.Equal(t, test.expectedVersion, version)
			assert.Equal(t, test.expectedRepoURL, repoURL)
		})
	}
}
