package kuberneteshelmcasting

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// One values file per kind, under the kind's own directory, so a casting file
// carrying both kinds pours a release's values per document.
func TestForge(t *testing.T) {
	for _, test := range []struct {
		name         string
		kind         collectionagent.CollectorKind
		expectedPath string
	}{
		{"Agent_Valid", collectionagent.CollectorKindAgent, "collectionagent/collector/agent/values.yaml"},
		{"Deployment_Valid", collectionagent.CollectorKindDeployment, "collectionagent/collector/deployment/values.yaml"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := moldedCasting(t, test.kind, defaultEnv())

			p := pourer.New("collectionagent")
			require.NoError(t, New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *config, p))

			materials, err := p.Pour()
			require.NoError(t, err)
			require.Len(t, materials, 1)

			assert.Equal(t, test.expectedPath, materials[0].Path())
			assert.NotEmpty(t, materials[0].FmtContents())
		})
	}
}

func TestReleaseName(t *testing.T) {
	for _, test := range []struct {
		name                string
		metadataName        string
		kind                collectionagent.CollectorKind
		expectedReleaseName string
	}{
		{"Agent_Valid", "signoz", collectionagent.CollectorKindAgent, "signoz-collector-agent"},
		{"Deployment_Valid", "signoz", collectionagent.CollectorKindDeployment, "signoz-collector-deployment"},
		{"RenamedAgent_Valid", "acme", collectionagent.CollectorKindAgent, "acme-collector-agent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Metadata.Name = test.metadataName
			config.Spec.Collector.Kind = test.kind

			assert.Equal(t, test.expectedReleaseName, releaseName(*config))
		})
	}
}

// Helm has no "latest" token, so the annotation's default and a stated "latest"
// both reach the SDK as the empty version that means the repository's newest.
func TestChartSource(t *testing.T) {
	for _, test := range []struct {
		name            string
		annotations     map[string]string
		expectedChart   string
		expectedVersion string
		expectedRepoURL string
	}{
		{"Unstated_Valid", nil, "k8s-infra", "", "https://charts.signoz.io"},
		{
			"StatedLatest_Valid",
			map[string]string{collectionagent.HelmChartVersion.Key: "latest"},
			"k8s-infra", "", "https://charts.signoz.io",
		},
		{
			"StatedVersion_Valid",
			map[string]string{collectionagent.HelmChartVersion.Key: "0.17.1"},
			"k8s-infra", "0.17.1", "https://charts.signoz.io",
		},
		{
			"StatedRepo_Valid",
			map[string]string{collectionagent.HelmChartRepoURL.Key: "https://charts.example.com"},
			"k8s-infra", "", "https://charts.example.com",
		},
		{
			"SlashedRef_Valid",
			map[string]string{collectionagent.HelmChart.Key: "./charts/k8s-infra"},
			"./charts/k8s-infra", "", "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Metadata.Annotations = test.annotations

			chart, version, repoURL := chartSource(*config)

			assert.Equal(t, test.expectedChart, chart)
			assert.Equal(t, test.expectedVersion, version)
			assert.Equal(t, test.expectedRepoURL, repoURL)
		})
	}
}
