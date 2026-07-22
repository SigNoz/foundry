package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrategicMergeYAML(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		override  string
		listTypes ListTypes
		pass      bool
		expected  string
	}{
		{
			name:     "PartialOverride_DeepMerges",
			base:     "display_name: cluster\nlogger:\n  level: information\n  size: 1000M\n",
			override: "logger:\n  level: debug\n",
			pass:     true,
			expected: "display_name: cluster\nlogger:\n  level: debug\n  size: 1000M\n",
		},
		{
			name:     "NullOverride_DeletesKey",
			base:     "logger:\n  level: information\n  size: 1000M\n",
			override: "logger:\n  size: null\n",
			pass:     true,
			expected: "logger:\n  level: information\n",
		},
		{
			name:     "UndeclaredList_Replaced",
			base:     "receivers:\n- otlp\n",
			override: "receivers:\n- docker_stats\n",
			pass:     true,
			expected: "receivers:\n- docker_stats\n",
		},
		{
			name:      "Set_UnionsBaseFirstDeduped",
			base:      "receivers:\n- otlp\n",
			override:  "receivers:\n- docker_stats\n- otlp\n",
			listTypes: ListTypes{"receivers": ListTypeSet},
			pass:      true,
			expected:  "receivers:\n- otlp\n- docker_stats\n",
		},
		{
			name:      "Set_RestoresDroppedBaseMember",
			base:      "receivers:\n- otlp\n",
			override:  "receivers:\n- docker_stats\n",
			listTypes: ListTypes{"receivers": ListTypeSet},
			pass:      true,
			expected:  "receivers:\n- otlp\n- docker_stats\n",
		},
		{
			name:      "Ordered_InsertsBeforeTerminal",
			base:      "processors:\n- memory_limiter\n- batch\n",
			override:  "processors:\n- resourcedetection\n",
			listTypes: ListTypes{"processors": ListTypeOrdered},
			pass:      true,
			expected:  "processors:\n- memory_limiter\n- resourcedetection\n- batch\n",
		},
		{
			name:      "WildcardPath_MatchesDeclaredBranchOnly",
			base:      "service:\n  pipelines:\n    metrics:\n      receivers:\n      - otlp\n    traces:\n      receivers:\n      - otlp\n",
			override:  "service:\n  pipelines:\n    metrics:\n      receivers:\n      - docker_stats\n",
			listTypes: ListTypes{"service.pipelines.*.receivers": ListTypeSet},
			pass:      true,
			expected:  "service:\n  pipelines:\n    metrics:\n      receivers:\n      - otlp\n      - docker_stats\n    traces:\n      receivers:\n      - otlp\n",
		},
		{
			name:      "MapListUnderSet_DegradesToAtomic",
			base:      "server:\n- hostname: a\n",
			override:  "server:\n- hostname: b\n",
			listTypes: ListTypes{"server": ListTypeSet},
			pass:      true,
			expected:  "server:\n- hostname: b\n",
		},
		{
			name:     "EmptyOverride_KeepsBase",
			base:     "logger:\n  level: information\n",
			override: "",
			pass:     true,
			expected: "logger:\n  level: information\n",
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
			merged, err := StrategicMergeYAML(tt.base, tt.override, tt.listTypes)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, merged)
		})
	}
}
