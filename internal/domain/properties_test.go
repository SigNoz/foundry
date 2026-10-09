package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPropertiesMerge(t *testing.T) {
	tests := []struct {
		name           string
		base           Properties
		other          Properties
		pass           bool
		expectedValues map[string]any
	}{
		{
			name:           "Value_Valid",
			base:           NewProperties().Set("mode", "docker"),
			other:          NewProperties().Set("mode", "kubernetes").Set("platform", ""),
			pass:           true,
			expectedValues: map[string]any{"mode": "kubernetes", "platform": ""},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := test.base.Merge(test.other).Map()
			if test.pass {
				assert.Equal(t, test.expectedValues, values)
				return
			}

			assert.NotEqual(t, test.expectedValues, values)
		})
	}
}
