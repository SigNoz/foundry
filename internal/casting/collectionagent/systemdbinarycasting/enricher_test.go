package systemdbinarycasting

import (
	"context"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrichStatus(t *testing.T) {
	testCases := []struct {
		name         string
		replicas     *int
		expectedCode int
		pass         bool
	}{
		{
			name:     "ReplicasOne_Valid",
			replicas: v1alpha1.IntPtr(1),
			pass:     true,
		},
		{
			name:     "ReplicasNil_Valid",
			replicas: nil,
			pass:     true,
		},
		{
			name:         "ReplicasTwo_Invalid",
			replicas:     v1alpha1.IntPtr(2),
			expectedCode: foundryerrors.TypeUnsupported.ExitCode(),
			pass:         false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Metadata.Name = "acme"
			config.Spec.Collector.Spec.Cluster.Replicas = testCase.replicas

			err := newSystemdBinaryMoldingEnricher(config).EnrichStatus(context.Background(), v1alpha1.MoldingKindCollector, config)

			if !testCase.pass {
				require.Error(t, err)
				assert.Equal(t, testCase.expectedCode, foundryerrors.ExitCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, config.Spec.Collector.Status.Config.Data[config.Spec.Collector.Kind.ConfigKey()])
		})
	}
}
