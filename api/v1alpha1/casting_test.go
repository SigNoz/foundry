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
			name: "explicit kind dispatches into the matching spec type",
			input: `{
				"apiVersion": "v1alpha1",
				"kind": "SigNoz",
				"metadata": {"name": "signoz"},
				"spec": {"deployment": {"mode": "docker", "flavor": "compose"}}
			}`,
			assertOK: func(t *testing.T, c Casting) {
				assert.Equal(t, KindSigNoz, c.Kind)
				assert.IsType(t, &SigNozCastingSpec{}, c.Spec)
				assert.IsType(t, &SigNozCastingStatus{}, c.Status)
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
				assert.IsType(t, &SigNozCastingSpec{}, c.Spec)
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

func TestJSONSchemaOneOfVariants(t *testing.T) {
	t.Parallel()

	variants := Casting{}.JSONSchemaOneOf()
	require.Len(t, variants, 1)
	assert.IsType(t, kindSigNozCasting{}, variants[0])
}
