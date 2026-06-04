package ledger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAIAgent(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name:     "Empty_Unknown",
			env:      map[string]string{},
			expected: "",
		},
		{
			name: "AIAgent_NormalizedToLeadingToken",
			env: map[string]string{
				"AI_AGENT":               "claude-code_2-1-161_agent",
				"CLAUDECODE":             "1",
				"CLAUDE_CODE_ENTRYPOINT": "cli",
			},
			expected: "claude",
		},
		{
			name: "AIAgent_CopilotCLIAlias",
			env: map[string]string{
				"AI_AGENT": "GitHub-Copilot-CLI",
			},
			expected: "github-copilot",
		},
		{
			name: "AIAgent_WinsOverMarkers",
			env: map[string]string{
				"AI_AGENT":   "cursor",
				"CLAUDECODE": "1",
			},
			expected: "cursor",
		},
		{
			name: "AIAgent_WhitespaceIgnored",
			env: map[string]string{
				"AI_AGENT":   "   ",
				"CLAUDECODE": "1",
			},
			expected: "claude",
		},
		{
			name: "MarkerTable_Claude",
			env: map[string]string{
				"CLAUDECODE": "1",
			},
			expected: "claude",
		},
		{
			name: "MarkerTable_CoworkBeforeClaude",
			env: map[string]string{
				"CLAUDE_CODE_IS_COWORK": "1",
				"CLAUDECODE":            "1",
			},
			expected: "cowork",
		},
		{
			name: "MarkerTable_Codex",
			env: map[string]string{
				"CODEX_THREAD_ID": "thread-1",
			},
			expected: "codex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getEnv := func(key string) string { return tt.env[key] }
			assert.Equal(t, tt.expected, AIAgent(getEnv))
		})
	}
}
