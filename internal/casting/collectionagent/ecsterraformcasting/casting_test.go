package ecsterraformcasting

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForge(t *testing.T) {
	for _, test := range []struct {
		name          string
		kind          collectionagent.CollectorKind
		replicas      *int
		pass          bool
		expectedCode  int
		expectedPaths []string
	}{
		{
			name:     "Sidecar_Valid",
			kind:     collectionagent.CollectorKindSidecar,
			replicas: v1alpha1.IntPtr(1),
			pass:     true,
			expectedPaths: []string{
				"collectionagent/versions.tf.json",
				"collectionagent/variables.tf.json",
				"collectionagent/main.tf.json",
				"collectionagent/outputs.tf.json",
				"collectionagent/collector/sidecar/sidecar.yaml",
			},
		},
		{
			name:     "Agent_Valid",
			kind:     collectionagent.CollectorKindAgent,
			replicas: v1alpha1.IntPtr(1),
			pass:     true,
			expectedPaths: []string{
				"collectionagent/versions.tf.json",
				"collectionagent/providers.tf.json",
				"collectionagent/backend.tf.json",
				"collectionagent/variables.tf.json",
				"collectionagent/terraform.tfvars.json",
				"collectionagent/main.tf.json",
				"collectionagent/collector.tf.json",
				"collectionagent/collector/agent/agent.yaml",
			},
		},
		{
			name:         "SidecarReplicas_Invalid",
			kind:         collectionagent.CollectorKindSidecar,
			replicas:     v1alpha1.IntPtr(2),
			expectedCode: foundryerrors.TypeInvalidInput.ExitCode(),
		},
		{
			name:         "Deployment_Invalid",
			kind:         collectionagent.CollectorKindDeployment,
			replicas:     v1alpha1.IntPtr(1),
			expectedCode: foundryerrors.TypeUnsupported.ExitCode(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Spec.Collector.Kind = test.kind
			config.Spec.Collector.Spec.Cluster.Replicas = test.replicas
			config.Spec.Collector.Spec.Config.Data = map[string]string{test.kind.ConfigKey(): "receivers: {}\n"}

			p := pourer.New("collectionagent")
			err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *config, p)

			if !test.pass {
				require.Error(t, err)
				assert.Equal(t, test.expectedCode, foundryerrors.ExitCode(err))

				return
			}

			require.NoError(t, err)

			materials, err := p.Pour()
			require.NoError(t, err)

			paths := make([]string, 0, len(materials))
			for _, material := range materials {
				paths = append(paths, material.Path())
			}

			assert.ElementsMatch(t, test.expectedPaths, paths)
		})
	}
}

// The module is imported into the operator's own root, which holds the
// provider, the state and the region.
func TestSidecarTemplateData(t *testing.T) {
	config := collectionagent.Default()
	config.Metadata.Name = "acme"
	config.Spec.Collector.Kind = collectionagent.CollectorKindSidecar

	data := sidecarTemplateDataFor(*config)

	assert.Equal(t, "acme-collector-sidecar", data.Container)
	assert.Equal(t, "/acme-collectionagent/collector/sidecar", data.Parameter)
	assert.Equal(t, "acme-collectionagent-parameter-read", data.Policy)
	assert.Equal(t, "collector/sidecar/sidecar.yaml", data.Source)
}
