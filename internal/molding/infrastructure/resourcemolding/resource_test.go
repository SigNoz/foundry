package resourcemolding

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestMoldV1Alpha1(t *testing.T) {
	tests := []struct {
		name     string
		kind     infrastructure.ResourceKind
		pass     bool
		expected infrastructure.ResourceConfig
	}{
		{
			name: "InstallationResource_PersistentAndEphemeralNodeGroups",
			kind: infrastructure.ResourceKindInstallation,
			pass: true,
			expected: infrastructure.ResourceConfig{
				NodeGroups: map[infrastructure.StorageClass]infrastructure.ResourceConfigNodeGroup{
					infrastructure.StorageClassPersistent: {
						MinSize:    v1alpha1.IntPtr(3),
						MaxSize:    v1alpha1.IntPtr(3),
						CPU:        v1alpha1.IntPtr(2),
						Memory:     v1alpha1.IntPtr(8),
						RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
						DataVolume: &infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(50)},
					},
					infrastructure.StorageClassEphemeral: {
						MinSize:    v1alpha1.IntPtr(1),
						MaxSize:    v1alpha1.IntPtr(1),
						CPU:        v1alpha1.IntPtr(2),
						Memory:     v1alpha1.IntPtr(4),
						RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
					},
				},
			},
		},
		{
			name: "CollectionAgentResource_EphemeralNodeGroup",
			kind: infrastructure.ResourceKindCollectionAgent,
			pass: true,
			expected: infrastructure.ResourceConfig{
				NodeGroups: map[infrastructure.StorageClass]infrastructure.ResourceConfigNodeGroup{
					infrastructure.StorageClassEphemeral: {
						MinSize:    v1alpha1.IntPtr(1),
						MaxSize:    v1alpha1.IntPtr(1),
						CPU:        v1alpha1.IntPtr(2),
						Memory:     v1alpha1.IntPtr(4),
						RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
					},
				},
			},
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

			got := infrastructure.ResourceConfig{}
			assert.NoError(t, domain.UnmarshalYAML([]byte(config.Spec.Resource.Status.Config.Data[ResourceConfigName]), &got))
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestMoldV1Alpha1_AddressesBaseline(t *testing.T) {
	config := infrastructure.Default()
	config.Spec.Resource.Kind = infrastructure.ResourceKindInstallation

	err := New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(context.Background(), config)
	assert.NoError(t, err)
	assert.Equal(t, []string{"tcp://0.0.0.0:4317", "tcp://0.0.0.0:4318"}, config.Spec.Resource.Status.Addresses.OTLP)
	assert.Equal(t, []string{"tcp://0.0.0.0:8080"}, config.Spec.Resource.Status.Addresses.APIServer)
}

func TestMoldV1Alpha1_PreservesEnricherContributions(t *testing.T) {
	config := infrastructure.Default()
	config.Spec.Resource.Kind = infrastructure.ResourceKindInstallation
	config.Spec.Resource.Status.Addresses.OTLP = []string{"tcp://0.0.0.0:9411"}
	config.Spec.Resource.Status.Config.Data = map[string]string{
		ResourceConfigName: `nodeGroups:
  persistent:
    minSize: 4
    maxSize: 4
    nodes: [{ordinal: 0}, {ordinal: 1}, {ordinal: 2}, {ordinal: 3}]
  ephemeral:
    minSize: 2
    maxSize: 2
`,
	}

	err := New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(context.Background(), config)
	assert.NoError(t, err)
	assert.Equal(t, []string{"tcp://0.0.0.0:4317", "tcp://0.0.0.0:4318", "tcp://0.0.0.0:9411"}, config.Spec.Resource.Status.Addresses.OTLP)

	doc := config.Spec.Resource.Status.Config.Data[ResourceConfigName]

	// Casting-specific keys survive the merge untouched.
	assert.Contains(t, doc, "ordinal")

	got := infrastructure.ResourceConfig{}
	assert.NoError(t, domain.UnmarshalYAML([]byte(doc), &got))

	// Groups are keyed by class, so the contribution states only the sizes it
	// changes and the baseline's other fields survive -- under plain document
	// merge, with no list strategy.
	assert.Len(t, got.NodeGroups, 2)

	persistent := got.NodeGroups[infrastructure.StorageClassPersistent]
	assert.Equal(t, v1alpha1.IntPtr(4), persistent.MinSize)
	assert.Equal(t, v1alpha1.IntPtr(8), persistent.Memory)
	assert.Equal(t, v1alpha1.IntPtr(50), persistent.DataVolume.Size)

	ephemeral := got.NodeGroups[infrastructure.StorageClassEphemeral]
	assert.Equal(t, v1alpha1.IntPtr(2), ephemeral.MinSize)
	assert.Equal(t, v1alpha1.IntPtr(4), ephemeral.Memory)
}

func TestMoldV1Alpha1_ContributionValidity(t *testing.T) {
	tests := []struct {
		name         string
		contribution string
		pass         bool
	}{
		{
			// A partial override is the point of keying by class: the baseline
			// supplies everything it does not mention.
			name:         "PartialOverride_Valid",
			contribution: "nodeGroups:\n  persistent:\n    minSize: 3\n",
			pass:         true,
		},
		{
			// An unnameable group cannot be reached by any consumer, so an
			// unknown key fails at unmarshal rather than provisioning nodes
			// nothing will be placed on.
			name:         "UnknownClass_Invalid",
			contribution: "nodeGroups:\n  keeper:\n    minSize: 3\n",
			pass:         false,
		},
		{
			name:         "PinnedGroupWithUnequalBounds_Invalid",
			contribution: "nodeGroups:\n  persistent:\n    minSize: 3\n    maxSize: 5\n",
			pass:         false,
		},
		{
			// A null deletes the key under RFC 7386, so this removes the
			// baseline's data volume from a class that requires one.
			name:         "PersistentWithoutDataVolume_Invalid",
			contribution: "nodeGroups:\n  persistent:\n    dataVolume: null\n",
			pass:         false,
		},
		{
			name:         "EphemeralWithDataVolume_Invalid",
			contribution: "nodeGroups:\n  ephemeral:\n    dataVolume:\n      size: 20\n",
			pass:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := infrastructure.Default()
			config.Spec.Resource.Kind = infrastructure.ResourceKindInstallation
			config.Spec.Resource.Status.Config.Data = map[string]string{ResourceConfigName: tt.contribution}

			err := New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(context.Background(), config)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
