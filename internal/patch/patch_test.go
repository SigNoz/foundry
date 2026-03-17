package patch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchTarget(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"exact match", "test.yaml", "test.yaml", true},
		{"full path match", "deployment/compose.yaml", "deployment/compose.yaml", true},
		{"basename does not match full path", "compose.yaml", "deployment/compose.yaml", false},
		{"glob match", "*.yaml", "test.yaml", true},
		{"glob no match", "*.json", "test.yaml", false},
		{"prefix glob", "clickhouse-*.yaml", "clickhouse-shard-0.yaml", true},
		{"path glob", "deployment/*.yaml", "deployment/compose.yaml", true},
		{"no match", "other.yaml", "test.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchTarget(tt.pattern, tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
