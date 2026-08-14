package yamlconfig

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetV1Alpha1(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		assert func(t *testing.T, casting installation.Casting)
	}{
		{
			name: "Defaults",
			input: `
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
`,
			assert: func(t *testing.T, casting installation.Casting) {
				// All moldings should be enabled by default
				assert.True(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.Spec.MetaStore.Spec.Enabled)
				assert.True(t, *casting.Spec.Ingester.Spec.Enabled)
				// The default telemetrykeeper kind carries its image and version
				assert.Equal(t, installation.TelemetryKeeperKindClickhouseKeeper, casting.Spec.TelemetryKeeper.Kind)
				assert.Equal(t, "clickhouse/clickhouse-keeper:25.12.5", casting.Spec.TelemetryKeeper.Spec.Image)
				assert.Equal(t, "25.12.5", casting.Spec.TelemetryKeeper.Spec.Version)
			},
		},
		{
			name: "TelemetryKeeperZookeeperKindDefaults",
			input: `
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  telemetrykeeper:
    kind: zookeeper
`,
			assert: func(t *testing.T, casting installation.Casting) {
				assert.Equal(t, installation.TelemetryKeeperKindZookeeper, casting.Spec.TelemetryKeeper.Kind)
				assert.Equal(t, "signoz/zookeeper:3.7.1", casting.Spec.TelemetryKeeper.Spec.Image)
				assert.Equal(t, "3.7.1", casting.Spec.TelemetryKeeper.Spec.Version)
			},
		},
		{
			name: "TelemetryKeeperZookeeperKindUserImageWins",
			input: `
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  telemetrykeeper:
    kind: zookeeper
    spec:
      image: signoz/zookeeper:3.8.4
      version: 3.8.4
`,
			assert: func(t *testing.T, casting installation.Casting) {
				assert.Equal(t, installation.TelemetryKeeperKindZookeeper, casting.Spec.TelemetryKeeper.Kind)
				assert.Equal(t, "signoz/zookeeper:3.8.4", casting.Spec.TelemetryKeeper.Spec.Image)
				assert.Equal(t, "3.8.4", casting.Spec.TelemetryKeeper.Spec.Version)
			},
		},
		{
			name: "DisableMetaStore",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.False(t, *casting.Spec.MetaStore.Spec.Enabled)
				// Other moldings should remain enabled
				assert.True(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.Spec.Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableSignoz",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.False(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.Spec.MetaStore.Spec.Enabled)
				assert.True(t, *casting.Spec.Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableIngester",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.False(t, *casting.Spec.Ingester.Spec.Enabled)
				assert.True(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.Spec.MetaStore.Spec.Enabled)
			},
		},
		{
			name: "DisableTelemetryStore",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.False(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.Spec.MetaStore.Spec.Enabled)
				assert.True(t, *casting.Spec.Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableTelemetryKeeper",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.False(t, *casting.Spec.TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.MetaStore.Spec.Enabled)
				assert.True(t, *casting.Spec.Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisableMultipleMoldings",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.False(t, *casting.Spec.MetaStore.Spec.Enabled)
				assert.False(t, *casting.Spec.TelemetryKeeper.Spec.Enabled)
				assert.True(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisabledWithOtherFields",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.False(t, *casting.Spec.MetaStore.Spec.Enabled)
				assert.Equal(t, "custom:1.0", casting.Spec.MetaStore.Spec.Image)
			},
		},
		{
			name: "ExplicitEnabledTrue",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.True(t, *casting.Spec.MetaStore.Spec.Enabled)
			},
		},
		{
			name: "OverrideImageKeepsEnabledDefault",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				// Enabled should remain true (default) when only image is overridden
				assert.True(t, *casting.Spec.MetaStore.Spec.Enabled)
				assert.Equal(t, "postgres:15", casting.Spec.MetaStore.Spec.Image)
			},
		},
		{
			name: "OverrideVersion",
			input: `
apiVersion: v1alpha1
kind: Installation
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
			assert: func(t *testing.T, casting installation.Casting) {
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.Equal(t, "24.8", casting.Spec.TelemetryStore.Spec.Version)
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

			cfg := New(slog.New(slog.DiscardHandler))
			castings, err := cfg.GetV1Alpha1(context.Background(), castingPath)
			require.NoError(t, err)
			require.Len(t, castings, 1)

			inst, ok := castings[0].(*installation.Casting)
			require.True(t, ok, "expected *installation.Casting, got %T", castings[0])
			tc.assert(t, *inst)
		})
	}
}

func TestGetV1Alpha1Merge(t *testing.T) {
	testCases := []struct {
		name     string
		base     installation.Casting
		override installation.Casting
		assert   func(t *testing.T, casting installation.Casting)
	}{
		{
			name: "EmptyOverride",
			base: *installation.Default(&installation.Casting{}),
			// A declared casting always carries its kind: the loader reads it
			// before the document reaches its type.
			override: installation.Casting{
				CastingMeta: v1alpha1.CastingMeta{Kind: v1alpha1.KindInstallation},
			},
			assert: func(t *testing.T, casting installation.Casting) {
				assert.True(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.MetaStore.Spec.Enabled)
				assert.True(t, *casting.Spec.Ingester.Spec.Enabled)
			},
		},
		{
			name: "DisabledMoldingOverride",
			base: *installation.Default(&installation.Casting{}),
			override: installation.Casting{
				CastingMeta: v1alpha1.CastingMeta{Kind: v1alpha1.KindInstallation},
				Spec: installation.Spec{
					MetaStore: installation.MetaStore{
						Spec: v1alpha1.MoldingSpec{
							Enabled: domain.NewBoolPtr(false),
						},
					},
				},
			},
			assert: func(t *testing.T, casting installation.Casting) {
				assert.False(t, *casting.Spec.MetaStore.Spec.Enabled)
				// Other moldings should remain enabled
				assert.True(t, *casting.Spec.Signoz.Spec.Enabled)
				assert.True(t, *casting.Spec.TelemetryStore.Spec.Enabled)
				assert.True(t, *casting.Spec.Ingester.Spec.Enabled)
			},
		},
	}

	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			base := tc.base
			override := tc.override
			err := v1alpha1.Merge(&base, &override)
			require.NoError(t, err)
			tc.assert(t, base)
		})
	}
}

func TestGetV1Alpha1Documents(t *testing.T) {
	installationDocument := `
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
`
	collectionAgentDocument := `
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz-agent
spec:
  deployment:
    mode: docker
    flavor: compose
`
	infrastructureDocument := `
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz-infra
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
`
	collectionAgentAgentDocument := `
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz-agent
spec:
  deployment:
    mode: docker
    flavor: compose
  collector:
    kind: agent
`

	tests := []struct {
		name          string
		contents      string
		expectedKinds []v1alpha1.Kind
		expectedError string
		pass          bool
	}{
		{
			name:          "SingleInstallation_Valid",
			contents:      installationDocument,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInstallation},
			pass:          true,
		},
		{
			name:          "TwoKinds_Valid",
			contents:      installationDocument + "---" + collectionAgentDocument,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInstallation, v1alpha1.KindCollectionAgent},
			pass:          true,
		},
		{
			// Cast order comes from the kind, never from where the document sits.
			name:          "TwoKindsReversed_Valid",
			contents:      collectionAgentDocument + "---" + installationDocument,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInstallation, v1alpha1.KindCollectionAgent},
			pass:          true,
		},
		{
			// Written in reverse: the order comes from v1alpha1.Kinds().
			name:          "EveryKindReversed_CastOrder_Valid",
			contents:      collectionAgentDocument + "---" + installationDocument + "---" + infrastructureDocument,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInfrastructure, v1alpha1.KindInstallation, v1alpha1.KindCollectionAgent},
			pass:          true,
		},
		{
			name:          "EmptyAndCommentDocuments_Valid",
			contents:      "# a note\n---" + installationDocument + "---\n\n---" + collectionAgentDocument,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInstallation, v1alpha1.KindCollectionAgent},
			pass:          true,
		},
		{
			name:          "LeadingSeparator_Valid",
			contents:      "---" + installationDocument,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInstallation},
			pass:          true,
		},
		{
			name:          "DuplicateKind_Invalid",
			contents:      installationDocument + "---" + installationDocument,
			expectedError: "Installation is declared twice",
		},
		{
			name:          "DuplicateCollectorKind_Invalid",
			contents:      collectionAgentAgentDocument + "---" + collectionAgentAgentDocument,
			expectedError: `CollectionAgent with collector kind "agent" is declared twice`,
		},
		{
			// An omitted collector kind defaults to agent.
			name:          "DuplicateCollectorKindByDefault_Invalid",
			contents:      collectionAgentDocument + "---" + collectionAgentAgentDocument,
			expectedError: `CollectionAgent with collector kind "agent" is declared twice`,
		},
		{
			name: "MissingKind_DefaultsToInstallation",
			contents: `
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
`,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInstallation},
			pass:          true,
		},
		{
			name: "UnknownKind_Invalid",
			contents: `
apiVersion: v1alpha1
kind: Substrate
metadata:
  name: signoz
`,
			expectedError: "invalid kind: Substrate",
		},
		{
			// The position names the document a reader counts, not only the ones
			// that carried a casting.
			name:          "SecondDocumentInvalid_ReportsItsPosition",
			contents:      installationDocument + "---\n# a note\n---\nkind: Substrate\n",
			expectedError: "document 3",
		},
		{
			name:          "NoDocuments_Invalid",
			contents:      "# nothing but a note\n",
			expectedError: "it declares no castings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			castingPath := filepath.Join(t.TempDir(), "casting.yaml")
			assert.NoError(t, os.WriteFile(castingPath, []byte(tt.contents), 0644))

			cfg := New(slog.New(slog.DiscardHandler))
			machineries, err := cfg.GetV1Alpha1(context.Background(), castingPath)
			if !tt.pass {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
				return
			}
			assert.NoError(t, err)

			kinds := make([]v1alpha1.Kind, 0, len(machineries))
			for _, machinery := range machineries {
				kinds = append(kinds, machinery.Kind())
			}
			assert.Equal(t, tt.expectedKinds, kinds)
		})
	}
}

func TestGetV1Alpha1Infrastructure(t *testing.T) {
	tests := []struct {
		name  string
		input string
		pass  bool
	}{
		{
			name: "Deployment_Valid",
			input: `
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
`,
			pass: true,
		},
		{
			name: "NameMissing_Invalid",
			input: `
apiVersion: v1alpha1
kind: Infrastructure
metadata: {}
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
`,
		},
		{
			name: "UnknownPlatform_Invalid",
			input: `
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
spec:
  deployment:
    platform: nowhere
    mode: ec2
    flavor: terraform
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			castingPath := filepath.Join(t.TempDir(), "casting.yaml")
			assert.NoError(t, os.WriteFile(castingPath, []byte(tt.input), 0644))

			cfg := New(slog.New(slog.DiscardHandler))
			machineries, err := cfg.GetV1Alpha1(context.Background(), castingPath)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, machineries, 1)
			if len(machineries) != 1 {
				return
			}

			casting, ok := machineries[0].(*infrastructure.Casting)
			assert.True(t, ok)
			if !ok {
				return
			}
			assert.Equal(t, v1alpha1.KindInfrastructure, casting.Kind())
		})
	}
}

func TestCreateV1Alpha1Lock(t *testing.T) {
	tests := []struct {
		name          string
		contents      string
		expectedKinds []v1alpha1.Kind
		expectedNames []string
	}{
		{
			name: "SingleDocument_RoundTrips",
			contents: `
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
`,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInstallation},
			expectedNames: []string{"signoz"},
		},
		{
			// Written agent-first, so the lock proves it records cast order.
			name: "TwoDocuments_RoundTripInCastOrder",
			contents: `
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz-agent
spec:
  deployment:
    mode: docker
    flavor: compose
---
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
`,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInstallation, v1alpha1.KindCollectionAgent},
			expectedNames: []string{"signoz", "signoz-agent"},
		},
		{
			// Written agent-first, so the lock proves it records cast order
			// across every kind.
			name: "EveryKind_RoundTripsInCastOrder",
			contents: `
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz-agent
spec:
  deployment:
    mode: docker
    flavor: compose
---
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
---
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz-infra
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
`,
			expectedKinds: []v1alpha1.Kind{v1alpha1.KindInfrastructure, v1alpha1.KindInstallation, v1alpha1.KindCollectionAgent},
			expectedNames: []string{"signoz-infra", "signoz", "signoz-agent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			castingPath := filepath.Join(t.TempDir(), "casting.yaml")
			assert.NoError(t, os.WriteFile(castingPath, []byte(tt.contents), 0644))

			cfg := New(slog.New(slog.DiscardHandler))

			machineries, err := cfg.GetV1Alpha1(ctx, castingPath)
			assert.NoError(t, err)
			assert.NoError(t, cfg.CreateV1Alpha1Lock(ctx, machineries, castingPath))

			locked, err := cfg.GetV1Alpha1Lock(ctx, castingPath)
			assert.NoError(t, err)

			kinds := make([]v1alpha1.Kind, 0, len(locked))
			names := make([]string, 0, len(locked))
			for _, machinery := range locked {
				kinds = append(kinds, machinery.Kind())
				names = append(names, machinery.Name())
			}

			assert.Equal(t, tt.expectedKinds, kinds)
			assert.Equal(t, tt.expectedNames, names)
		})
	}
}
