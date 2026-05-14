package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		base     TypeMetadata
		override TypeMetadata
		want     TypeMetadata
	}{
		{
			name:     "override Name wins over base",
			base:     TypeMetadata{Name: "base"},
			override: TypeMetadata{Name: "override"},
			want:     TypeMetadata{Name: "override"},
		},
		{
			name:     "override fills in unset base fields",
			base:     TypeMetadata{},
			override: TypeMetadata{Name: "fresh"},
			want:     TypeMetadata{Name: "fresh"},
		},
		{
			name:     "override Annotations (map with omitempty) does not clobber base when unset",
			base:     TypeMetadata{Name: "base", Annotations: map[string]string{"a": "1"}},
			override: TypeMetadata{Name: "base"},
			want:     TypeMetadata{Name: "base", Annotations: map[string]string{"a": "1"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := tc.base
			override := tc.override
			require.NoError(t, Merge(&base, &override))
			assert.Equal(t, tc.want, base)
		})
	}
}
