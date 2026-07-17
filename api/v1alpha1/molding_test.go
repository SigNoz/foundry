package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusData map[string]string // what the molding generated
		specData   map[string]string // the user's spec delta
		pass       bool
		expected   map[string]string
	}{
		{
			name:       "UserDelta_DeepMerge_SiblingsSurvive",
			statusData: map[string]string{"config-0-0.yaml": "logger:\n  level: information\nmacros:\n  replica: \"00\"\n  shard: \"00\"\n"},
			specData:   map[string]string{"config-0-0.yaml": "macros:\n  replica: example01-01-1\n"},
			pass:       true,
			expected:   map[string]string{"config-0-0.yaml": "logger:\n  level: information\nmacros:\n  replica: example01-01-1\n  shard: \"00\"\n"},
		},
		{
			name:       "NewFileDelta_AddedAsIs",
			statusData: map[string]string{"config-0-0.yaml": "a: b\n"},
			specData:   map[string]string{"extra.yaml": "foo: bar\n"},
			pass:       true,
			expected:   map[string]string{"config-0-0.yaml": "a: b\n", "extra.yaml": "foo: bar\n"},
		},
		{
			name:       "MalformedOverride_Invalid",
			statusData: map[string]string{"config-0-0.yaml": "a: b\n"},
			specData:   map[string]string{"config-0-0.yaml": "a: [b\n"},
			pass:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := MoldingSpec{Config: TypeConfig{Data: tt.specData}}
			status := MoldingStatus{Config: TypeConfig{Data: tt.statusData}}

			err := spec.MergeStatus(status)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, spec.Config.Data)
		})
	}
}
