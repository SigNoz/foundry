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
			name:     "MissingScalar_Copied",
			base:     NewProperties(),
			other:    NewProperties().Set("platform", "docker"),
			pass:     true,
			expected: map[string]any{"platform": "docker"},
		},
		{
			name:     "ExistingScalar_Replaced",
			base:     NewProperties().Set("mode", "docker"),
			other:    NewProperties().Set("mode", "kubernetes"),
			pass:     true,
			expected: map[string]any{"mode": "kubernetes"},
		},
		{
			name:     "DistinctKinds_Appended",
			base:     NewProperties().Set("kinds", []string{"Installation"}),
			other:    NewProperties().Set("kinds", []string{"CollectionAgent"}),
			pass:     true,
			expected: map[string]any{"kinds": []string{"Installation", "CollectionAgent"}},
		},
		{
			name:     "RepeatedKind_NotDuplicated",
			base:     NewProperties().Set("kinds", []string{"Installation", "CollectionAgent"}),
			other:    NewProperties().Set("kinds", []string{"CollectionAgent"}),
			pass:     true,
			expected: map[string]any{"kinds": []string{"Installation", "CollectionAgent"}},
		},
		{
			name:     "ListIntoMissingKey_Copied",
			base:     NewProperties().Set("platform", "docker"),
			other:    NewProperties().Set("kinds", []string{"Infrastructure"}),
			pass:     true,
			expected: map[string]any{"platform": "docker", "kinds": []string{"Infrastructure"}},
		},
		{
			name: "ScalarsAndKindsTogether_Merged",
			base: NewProperties().
				Set("kinds", []string{"Installation"}).
				Set("platform", "").
				Set("mode", "docker").
				Set("flavor", "compose").
				Set("patches_count", 0).
				Set("infrastructure_bound", false).
				Set("metastore_kind", "postgres").
				Set("telemetrystore_kind", "clickhouse").
				Set("telemetrykeeper_kind", "clickhousekeeper").
				Set("mcp_enabled", false),
			other: NewProperties().
				Set("kinds", []string{"CollectionAgent"}).
				Set("collectionagent_agent_platform", "").
				Set("collectionagent_agent_mode", "docker").
				Set("collectionagent_agent_flavor", "compose").
				Set("collectionagent_agent_patches_count", 1),
			pass: true,
			expected: map[string]any{
				"kinds":                               []string{"Installation", "CollectionAgent"},
				"platform":                            "",
				"mode":                                "docker",
				"flavor":                              "compose",
				"patches_count":                       0,
				"infrastructure_bound":                false,
				"metastore_kind":                      "postgres",
				"telemetrystore_kind":                 "clickhouse",
				"telemetrykeeper_kind":                "clickhousekeeper",
				"mcp_enabled":                         false,
				"collectionagent_agent_platform":      "",
				"collectionagent_agent_mode":          "docker",
				"collectionagent_agent_flavor":        "compose",
				"collectionagent_agent_patches_count": 1,
			},
		},
		{
			name:     "RepeatedKind_Duplicated",
			base:     NewProperties().Set("kinds", []string{"Installation", "CollectionAgent"}),
			other:    NewProperties().Set("kinds", []string{"CollectionAgent"}),
			pass:     false,
			expected: map[string]any{"kinds": []string{"Installation", "CollectionAgent", "CollectionAgent"}},
		},
		{
			name:     "ExistingScalar_Kept",
			base:     NewProperties().Set("mode", "docker"),
			other:    NewProperties().Set("mode", "kubernetes"),
			pass:     false,
			expected: map[string]any{"mode": "docker"},
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
