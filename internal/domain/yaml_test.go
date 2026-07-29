package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeYAML(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		override string
		pass     bool
		expected string
	}{
		{
			name:     "PartialOverride_Valid",
			base:     "logger:\n  level: information\n  size: 1000M\n",
			override: "logger:\n  level: debug\n",
			pass:     true,
			expected: "logger:\n  level: debug\n  size: 1000M\n",
		},
		{
			name:     "MalformedOverride_Invalid",
			base:     "a: b\n",
			override: "a: [b\n",
			pass:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := MergeYAML(tt.base, tt.override)

			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, merged)
		})
	}
}
