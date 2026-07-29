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
					{Name: "persistent", Persistent: v1alpha1.BoolPtr(true), Count: v1alpha1.IntPtr(3), VCPUs: v1alpha1.IntPtr(2), Memory: v1alpha1.IntPtr(8), Disk: v1alpha1.IntPtr(50)},
					{Name: "ephemeral", Persistent: v1alpha1.BoolPtr(false), Count: v1alpha1.IntPtr(1), VCPUs: v1alpha1.IntPtr(2), Memory: v1alpha1.IntPtr(4), Disk: v1alpha1.IntPtr(20)},
				},
			},
		},
		{
			name: "CollectionAgentResource_EphemeralNodeGroup",
			kind: infrastructure.ResourceKindCollectionAgent,
			pass: true,
			expected: infrastructure.ResourceConfig{
				NodeGroups: []infrastructure.ResourceConfigNodeGroup{
					{Name: "ephemeral", Persistent: v1alpha1.BoolPtr(false), Count: v1alpha1.IntPtr(1), VCPUs: v1alpha1.IntPtr(2), Memory: v1alpha1.IntPtr(4), Disk: v1alpha1.IntPtr(20)},
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
  persistent: true
  count: 4
  vcpus: 2
  memory: 8
  disk: 50
  nodes: [{ordinal: 0}, {ordinal: 1}, {ordinal: 2}, {ordinal: 3}]
- name: ephemeral
  persistent: false
  count: 2
  vcpus: 2
  memory: 4
  disk: 20
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

	// The contribution owns the node groups it states: the list replaces the
	// baseline wholesale.
	assert.Len(t, got.NodeGroups, 2)
	for _, group := range got.NodeGroups {
		switch group.Name {
		case "persistent":
			assert.Equal(t, v1alpha1.BoolPtr(true), group.Persistent)
			assert.Equal(t, v1alpha1.IntPtr(4), group.Count)
		case "ephemeral":
			assert.Equal(t, v1alpha1.BoolPtr(false), group.Persistent)
			assert.Equal(t, v1alpha1.IntPtr(2), group.Count)
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
