package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCastingUnmarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		assertOK  func(t *testing.T, c Casting)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "SigNoz kind decodes into *SigNozCastingSpec",
			input: `{
				"apiVersion": "v1alpha1",
				"kind": "SigNoz",
				"metadata": {"name": "signoz"},
				"spec": {"deployment": {"mode": "docker", "flavor": "compose"}}
			}`,
			assertOK: func(t *testing.T, c Casting) {
				assert.Equal(t, KindSigNoz, c.Kind)
				spec := c.SigNozSpec()
				assert.Equal(t, "docker", spec.Deployment.Mode.String())
				assert.Equal(t, "compose", spec.Deployment.Flavor.String())
				assert.NotNil(t, c.SigNozStatus())
			},
		},
		{
			name: "missing kind defaults to KindSigNoz",
			input: `{
				"apiVersion": "v1alpha1",
				"metadata": {"name": "signoz"},
				"spec": {"deployment": {"mode": "docker", "flavor": "compose"}}
			}`,
			assertOK: func(t *testing.T, c Casting) {
				assert.Equal(t, KindSigNoz, c.Kind)
				assert.NotNil(t, c.SigNozSpec())
			},
		},
		{
			name: "empty status decodes to empty pointer",
			input: `{
				"apiVersion": "v1alpha1",
				"kind": "SigNoz",
				"metadata": {"name": "signoz"},
				"spec": {"deployment": {"mode": "docker", "flavor": "compose"}}
			}`,
			assertOK: func(t *testing.T, c Casting) {
				assert.Equal(t, "", c.SigNozStatus().Checksum)
			},
		},
		{
			name: "unknown kind returns error",
			input: `{
				"apiVersion": "v1alpha1",
				"kind": "Unknown",
				"metadata": {"name": "signoz"},
				"spec": {}
			}`,
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid kind")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Casting
			err := json.Unmarshal([]byte(tc.input), &c)

			if tc.assertErr != nil {
				tc.assertErr(t, err)
				return
			}

			require.NoError(t, err)
			tc.assertOK(t, c)
		})
	}
}

func TestCastingMutationPropagatesThroughPointer(t *testing.T) {
	t.Parallel()

	input := `{
		"apiVersion": "v1alpha1",
		"kind": "SigNoz",
		"metadata": {"name": "signoz"},
		"spec": {"deployment": {"mode": "docker", "flavor": "compose"}}
	}`

	var c Casting
	require.NoError(t, json.Unmarshal([]byte(input), &c))

	c.SigNozSpec().MetaStore.Kind = MetaStoreKindPostgres
	assert.Equal(t, MetaStoreKindPostgres, c.SigNozSpec().MetaStore.Kind, "mutation through pointer should propagate")
}

func TestDefaultCastingShape(t *testing.T) {
	t.Parallel()

	c := DefaultCasting()
	assert.Equal(t, KindSigNoz, c.Kind)
	assert.Equal(t, "v1alpha1", c.APIVersion)
	assert.NotNil(t, c.SigNozSpec())
	assert.NotNil(t, c.SigNozStatus())
}

func TestExampleCastingShape(t *testing.T) {
	t.Parallel()

	c := ExampleCasting()
	assert.Equal(t, KindSigNoz, c.Kind)
	assert.NotNil(t, c.SigNozSpec())
	assert.Nil(t, c.Status, "ExampleCasting omits Status so YAML serialization stays minimal")
}

func TestJSONSchemaOneOfVariants(t *testing.T) {
	t.Parallel()

	variants := Casting{}.JSONSchemaOneOf()
	require.Len(t, variants, 1)
	_, ok := variants[0].(kindSigNozCasting)
	assert.True(t, ok, "expected kindSigNozCasting variant, got %T", variants[0])
}
