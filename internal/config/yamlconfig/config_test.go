package yamlconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetV1Alpha1(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		assert func(t *testing.T, casting v1alpha1.Casting)
	}{
		{
			name: "Defaults",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				// All moldings should be enabled by default
				assert.True(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableMetaStore",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  metastore:
    spec:
      enabled: false
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.False(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				// Other moldings should remain enabled
				assert.True(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableSignoz",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  signoz:
    spec:
      enabled: false
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.False(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableIngester",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  ingester:
    spec:
      enabled: false
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.False(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
			},
		},
		{
			name: "DisableTelemetryStore",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  telemetrystore:
    spec:
      enabled: false
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.False(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableTelemetryKeeper",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  telemetrykeeper:
    spec:
      enabled: false
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.False(t, *casting.SigNozSpec().TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableMultipleMoldings",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  metastore:
    spec:
      enabled: false
  telemetrykeeper:
    spec:
      enabled: false
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.False(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				assert.False(t, *casting.SigNozSpec().TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisabledWithOtherFields",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  metastore:
    spec:
      enabled: false
      image: custom:1.0
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.False(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				assert.Equal(t, "custom:1.0", casting.SigNozSpec().MetaStore.Spec.Image)
			},
		},
		{
			name: "ExplicitEnabledTrue",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  metastore:
    spec:
      enabled: true
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.True(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
			},
		},
		{
			name: "OverrideImageKeepsEnabledDefault",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  metastore:
    spec:
      image: postgres:15
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				// Enabled should remain true (default) when only image is overridden
				assert.True(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				assert.Equal(t, "postgres:15", casting.SigNozSpec().MetaStore.Spec.Image)
			},
		},
		{
			name: "OverrideVersion",
			input: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  telemetrystore:
    spec:
      version: "24.8"
`,
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.Equal(t, "24.8", casting.SigNozSpec().TelemetryStore.Spec.Version)
			},
		},
	}

	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			castingPath := filepath.Join(dir, "casting.yaml")
			err := os.WriteFile(castingPath, []byte(tc.input), 0644)
			require.NoError(t, err)

			cfg := New()
			casting, err := cfg.GetV1Alpha1(context.Background(), castingPath)
			require.NoError(t, err)

			tc.assert(t, casting)
		})
	}
}

func TestGetV1Alpha1Merge(t *testing.T) {
	testCases := []struct {
		name     string
		base     v1alpha1.Casting
		override v1alpha1.Casting
		assert   func(t *testing.T, casting v1alpha1.Casting)
	}{
		{
			name:     "EmptyOverride",
			base:     v1alpha1.DefaultSigNozCasting(),
			override: v1alpha1.Casting{},
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.True(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisabledMoldingOverride",
			base: v1alpha1.DefaultSigNozCasting(),
			override: v1alpha1.Casting{
				Spec: &v1alpha1.SigNozCastingSpec{
					MetaStore: v1alpha1.MetaStore{
						Spec: v1alpha1.MoldingSpec{
							Enabled: domain.NewBoolPtr(false),
						},
					},
				},
			},
			assert: func(t *testing.T, casting v1alpha1.Casting) {
				assert.False(t, *casting.SigNozSpec().MetaStore.Spec.Enabled)
				// Other moldings should remain enabled
				assert.True(t, *casting.SigNozSpec().Signoz.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.SigNozSpec().Ingester.Spec.Enabled)
			},
		},
	}

	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			base := tc.base
			override := tc.override
			err := v1alpha1.MergeCasting(&base, &override)
			require.NoError(t, err)
			tc.assert(t, base)
		})
	}
}
