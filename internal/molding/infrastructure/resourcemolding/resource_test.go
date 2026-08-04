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
				NodeGroups: []infrastructure.ResourceConfigNodeGroup{
					{
						Name:       "persistent",
						Storage:    infrastructure.StorageClassPersistent,
						MinSize:    v1alpha1.IntPtr(3),
						MaxSize:    v1alpha1.IntPtr(3),
						CPU:        v1alpha1.IntPtr(2),
						Memory:     v1alpha1.IntPtr(8),
						RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
						DataVolume: &infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(50)},
					},
					{
						Name:       "ephemeral",
						Storage:    infrastructure.StorageClassEphemeral,
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
				NodeGroups: []infrastructure.ResourceConfigNodeGroup{
					{
						Name:       "ephemeral",
						Storage:    infrastructure.StorageClassEphemeral,
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
- name: persistent
  minSize: 4
  maxSize: 4
  nodes: [{ordinal: 0}, {ordinal: 1}, {ordinal: 2}, {ordinal: 3}]
- name: ephemeral
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

	// Node groups merge by name: the contribution states only the sizes it
	// changes and the baseline's other fields survive.
	assert.Len(t, got.NodeGroups, 2)
	for _, group := range got.NodeGroups {
		switch group.Name {
		case "persistent":
			assert.Equal(t, v1alpha1.IntPtr(4), group.MinSize)
			assert.Equal(t, infrastructure.StorageClassPersistent, group.Storage)
			assert.Equal(t, v1alpha1.IntPtr(8), group.Memory)
			assert.Equal(t, v1alpha1.IntPtr(50), group.DataVolume.Size)
		case "ephemeral":
			assert.Equal(t, v1alpha1.IntPtr(2), group.MinSize)
			assert.Equal(t, infrastructure.StorageClassEphemeral, group.Storage)
			assert.Equal(t, v1alpha1.IntPtr(4), group.Memory)
		default:
			t.Fatalf("unexpected node group %q", group.Name)
		}
	}
}

func TestMoldV1Alpha1_IncompleteContributionFails(t *testing.T) {
	config := infrastructure.Default()
	config.Spec.Resource.Kind = infrastructure.ResourceKindInstallation
	config.Spec.Resource.Status.Config.Data = map[string]string{
		ResourceConfigName: "nodeGroups:\n- name: keeper\n  persistent: true\n  count: 3\n",
	}

	err := New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(context.Background(), config)
	assert.Error(t, err)
}
