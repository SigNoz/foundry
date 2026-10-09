package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPropertiesMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     Properties
		other    Properties
		pass     bool
		expected map[string]any
	}{
		{
			name:     "MissingKey_Copied",
			base:     NewProperties(),
			other:    NewProperties().Set("platform", "docker"),
			pass:     true,
			expected: map[string]any{"platform": "docker"},
		},
		{
			name:     "ExistingKey_Replaced",
			base:     NewProperties().Set("mode", "docker"),
			other:    NewProperties().Set("mode", "kubernetes"),
			pass:     true,
			expected: map[string]any{"mode": "kubernetes"},
		},
		{
			name:     "ExistingInt_Summed",
			base:     NewProperties().Set("kind_collectionagent_count", 1),
			other:    NewProperties().Set("kind_collectionagent_count", 1),
			pass:     true,
			expected: map[string]any{"kind_collectionagent_count": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := tt.base.Merge(tt.other).Map()
			if !tt.pass {
				assert.NotEqual(t, tt.expected, merged)
				return
			}

			assert.Equal(t, tt.expected, merged)
		})
	}
}
