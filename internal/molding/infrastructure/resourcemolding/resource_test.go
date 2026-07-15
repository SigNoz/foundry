package resourcemolding

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestMoldV1Alpha1(t *testing.T) {
	tests := []struct {
		name         string
		kind         infrastructure.ResourceKind
		expectedOTLP []string
		expectedUI   []string
		pass         bool
	}{
		{
			name:         "InstallationResource_ExposesOtlpAndUi",
			kind:         infrastructure.ResourceKindInstallation,
			expectedOTLP: []string{":4317", ":4318"},
			expectedUI:   []string{":8080"},
			pass:         true,
		},
		{
			name:         "CollectionAgentResource_ExposesOtlpOnly",
			kind:         infrastructure.ResourceKindCollectionAgent,
			expectedOTLP: []string{":4317", ":4318"},
			pass:         true,
		},
		{
			name: "UnknownResourceKind_Unsupported",
			kind: infrastructure.ResourceKind{},
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := infrastructure.Default()
			config.Spec.Resource.Kind = tt.kind

			err := New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(context.Background(), config)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOTLP, config.Spec.Resource.Status.Addresses.OTLP)
			assert.Equal(t, tt.expectedUI, config.Spec.Resource.Status.Addresses.UI)
		})
	}
}
