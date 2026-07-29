package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeConfigSet(t *testing.T) {
	tests := []struct {
		name     string
		seed     map[string]string
		path     string
		content  string
		expected string
	}{
		{
			name:     "NilData_Allocates",
			path:     "collector/agent/agent.yaml",
			content:  "a: 1",
			expected: "a: 1",
		},
		{
			name:     "SamePath_Overwrites",
			seed:     map[string]string{"collector/agent/agent.yaml": "a: 1"},
			path:     "collector/agent/agent.yaml",
			content:  "a: 2",
			expected: "a: 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TypeConfig{Data: tt.seed}
			config.Set(tt.path, []byte(tt.content))

			assert.Equal(t, tt.expected, config.Data[tt.path])
		})
	}
}
