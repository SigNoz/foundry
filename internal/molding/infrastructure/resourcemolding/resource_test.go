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
			name: "InstallationResource_PersistentStorageAndDefaultNodeGroup",
			kind: infrastructure.ResourceKindInstallation,
			pass: true,
			expected: infrastructure.ResourceConfig{
				Storage: infrastructure.ResourceConfigStorage{Persistent: v1alpha1.BoolPtr(true)},
				NodeGroups: []infrastructure.ResourceConfigNodeGroup{
					{Name: "default", Count: v1alpha1.IntPtr(2), VCPUs: v1alpha1.IntPtr(2), Memory: v1alpha1.IntPtr(8), Disk: v1alpha1.IntPtr(50)},
				},
			},
		},
		{
			name: "CollectionAgentResource_EphemeralStorageAndDefaultNodeGroup",
			kind: infrastructure.ResourceKindCollectionAgent,
			pass: true,
			expected: infrastructure.ResourceConfig{
				Storage: infrastructure.ResourceConfigStorage{Persistent: v1alpha1.BoolPtr(false)},
				NodeGroups: []infrastructure.ResourceConfigNodeGroup{
					{Name: "default", Count: v1alpha1.IntPtr(1), VCPUs: v1alpha1.IntPtr(2), Memory: v1alpha1.IntPtr(4), Disk: v1alpha1.IntPtr(20)},
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
		ResourceConfigName: "nodeGroups:\n- name: default\n  count: 4\n- name: keeper\n  count: 3\n  vcpus: 2\n  memory: 8\n  disk: 100\n",
	}

	err := New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(context.Background(), config)
	assert.NoError(t, err)
	assert.Equal(t, []string{"tcp://0.0.0.0:4317", "tcp://0.0.0.0:4318", "tcp://0.0.0.0:9411"}, config.Spec.Resource.Status.Addresses.OTLP)

	got := infrastructure.ResourceConfig{}
	assert.NoError(t, domain.UnmarshalYAML([]byte(config.Spec.Resource.Status.Config.Data[ResourceConfigName]), &got))

	assert.Equal(t, v1alpha1.BoolPtr(true), got.Storage.Persistent)
	assert.Len(t, got.NodeGroups, 2)
	for _, group := range got.NodeGroups {
		switch group.Name {
		case "default":
			assert.Equal(t, v1alpha1.IntPtr(4), group.Count)
			assert.Equal(t, v1alpha1.IntPtr(2), group.VCPUs)
			assert.Equal(t, v1alpha1.IntPtr(8), group.Memory)
			assert.Equal(t, v1alpha1.IntPtr(50), group.Disk)
		case "keeper":
			assert.Equal(t, v1alpha1.IntPtr(3), group.Count)
			assert.Equal(t, v1alpha1.IntPtr(100), group.Disk)
		default:
			t.Fatalf("unexpected node group %q", group.Name)
		}
	}
}
