package v1alpha1_test

import (
	"testing"

	v1alpha1 "github.com/signoz/foundry/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusData map[string]string // what the molding generated
		specData   map[string]string // the user's spec delta
		want       map[string]string
	}{
		{
			name:       "user delta deep-merges onto generated config, siblings survive",
			statusData: map[string]string{"config-0-0.yaml": "logger:\n  level: information\nmacros:\n  replica: \"00\"\n  shard: \"00\"\n"},
			specData:   map[string]string{"config-0-0.yaml": "macros:\n  replica: example01-01-1\n"},
			want:       map[string]string{"config-0-0.yaml": "logger:\n  level: information\nmacros:\n  replica: example01-01-1\n  shard: \"00\"\n"},
		},
		{
			name:       "delta for a file the molding did not generate is added as-is",
			statusData: map[string]string{"config-0-0.yaml": "a: b\n"},
			specData:   map[string]string{"extra.yaml": "foo: bar\n"},
			want:       map[string]string{"config-0-0.yaml": "a: b\n", "extra.yaml": "foo: bar\n"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := v1alpha1.MoldingSpec{Config: v1alpha1.TypeConfig{Data: tc.specData}}
			status := v1alpha1.MoldingStatus{Config: v1alpha1.TypeConfig{Data: tc.statusData}}

			require.NoError(t, spec.MergeStatus(status))
			assert.Equal(t, tc.want, spec.Config.Data)
		})
	}
}
