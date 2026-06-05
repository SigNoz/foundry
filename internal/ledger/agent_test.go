package ledger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentProperties(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected map[string]string
	}{
		{
			name:     "Empty_Unknown",
			env:      map[string]string{},
			expected: map[string]string{"invoked_by": "unknown"},
		},
		{
			name: "AIAgent_NormalizedToLeadingToken",
			env: map[string]string{
				"AI_AGENT":   "claude-code_2-1-161_agent",
				"CLAUDECODE": "1",
			},
			expected: map[string]string{
				"invoked_by":     "agent",
				"agent_name":     "claude",
				"agent_fullname": "claude-code_2-1-161_agent",
			},
		},
		{
			name: "AIAgent_WinsOverMarkers",
			env: map[string]string{
				"AI_AGENT":   "cursor",
				"CLAUDECODE": "1",
			},
			expected: map[string]string{
				"invoked_by":     "agent",
				"agent_name":     "cursor",
				"agent_fullname": "cursor",
			},
		},
		{
			name: "MarkerTable_Claude",
			env: map[string]string{
				"CLAUDECODE": "1",
			},
			expected: map[string]string{
				"invoked_by":     "agent",
				"agent_name":     "claude",
				"agent_fullname": "claude",
			},
		},
		{
			name: "MarkerTable_CoworkBeforeClaude",
			env: map[string]string{
				"CLAUDE_CODE_IS_COWORK": "1",
				"CLAUDECODE":            "1",
			},
			expected: map[string]string{
				"invoked_by":     "agent",
				"agent_name":     "cowork",
				"agent_fullname": "cowork",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Empty out every variable the detection reads so the
			// environment running the tests (e.g. an AI agent session)
			// cannot leak in, then apply the case's environment.
			t.Setenv("AI_AGENT", "")
			for _, d := range agentDetectors {
				for _, envVar := range d.envs {
					t.Setenv(envVar, "")
				}
			}
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			assert.Equal(t, tt.expected, NewAgentProperties())
		})
	}
}
