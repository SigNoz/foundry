package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerge_NilStrategyDeepMergesReplaceLists(t *testing.T) {
	base := map[string]any{
		"a": map[string]any{"x": 1, "y": 2},
		"l": []any{"keep"},
		"s": "base",
	}
	src := map[string]any{
		"a": map[string]any{"y": 3, "z": 4}, // nested map merges
		"l": []any{"override"},              // list replaced (no strategy)
		"n": "new",                          // new key added
	}

	Merge(base, src, nil)

	assert.Equal(t, map[string]any{"x": 1, "y": 3, "z": 4}, base["a"])
	assert.Equal(t, []any{"override"}, base["l"])
	assert.Equal(t, "base", base["s"])
	assert.Equal(t, "new", base["n"])
}

func TestMerge_ListStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy ListMerge
		base     []any
		override []any
		expected []any
	}{
		{"Replace_Default", ListMergeReplace, []any{"otlp"}, []any{"docker_stats"}, []any{"docker_stats"}},
		{"Set_UnionsBaseFirstDeduped", ListMergeSet, []any{"otlp"}, []any{"docker_stats", "otlp"}, []any{"otlp", "docker_stats"}},
		{"Set_RestoresDroppedBaseMember", ListMergeSet, []any{"otlp"}, []any{"docker_stats"}, []any{"otlp", "docker_stats"}},
		{"Ordered_InsertsBeforeTerminal", ListMergeOrdered,
			[]any{"memory_limiter", "resourcedetection", "batch"}, []any{"k8sattributes"},
			[]any{"memory_limiter", "resourcedetection", "k8sattributes", "batch"}},
		{"Ordered_EmptyOverrideKeepsBase", ListMergeOrdered,
			[]any{"memory_limiter", "resourcedetection", "batch"}, nil,
			[]any{"memory_limiter", "resourcedetection", "batch"}},
		{"Set_ListOfMaps_DegradesToReplace", ListMergeSet,
			[]any{map[string]any{"name": "a"}}, []any{map[string]any{"name": "b"}},
			[]any{map[string]any{"name": "b"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := map[string]any{"list": tt.base}
			override := map[string]any{"list": tt.override}
			Merge(base, override, map[string]ListMerge{"list": tt.strategy})
			assert.Equal(t, tt.expected, base["list"])
		})
	}
}

func TestMerge_WildcardPathAndUntouchedBranches(t *testing.T) {
	base := map[string]any{
		"service": map[string]any{
			"pipelines": map[string]any{
				"metrics": map[string]any{"receivers": []any{"otlp"}},
				"traces":  map[string]any{"receivers": []any{"otlp"}},
			},
		},
	}
	override := map[string]any{
		"service": map[string]any{
			"pipelines": map[string]any{
				"metrics": map[string]any{"receivers": []any{"docker_stats"}},
			},
		},
	}

	Merge(base, override, map[string]ListMerge{"service.pipelines.*.receivers": ListMergeSet})

	pipelines := base["service"].(map[string]any)["pipelines"].(map[string]any)
	// wildcard matched metrics → union restores otlp
	assert.Equal(t, []any{"otlp", "docker_stats"}, pipelines["metrics"].(map[string]any)["receivers"])
	// traces untouched by the override → base kept
	assert.Equal(t, []any{"otlp"}, pipelines["traces"].(map[string]any)["receivers"])
}
