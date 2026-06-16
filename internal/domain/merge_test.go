package domain_test

import (
	"testing"

	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeYAML(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		override string
		strategy map[string]domain.ListMerge
		expected string
	}{
		{
			name: "PartialOverride_DeepMerges",
			base: `logger:
  level: information
  count: 10
http_port: 8123`,
			override: `logger:
  level: debug`,
			expected: `http_port: 8123
logger:
  count: 10
  level: debug
`,
		},
		{
			name: "UnmatchedList_Replaced",
			base: `remote_servers:
  cluster:
    shard:
    - replica:
      - host: a
        port: 9000`,
			override: `remote_servers:
  cluster:
    shard:
    - replica:
      - host: b
        port: 9000`,
			strategy: map[string]domain.ListMerge{"remote_servers.*.shard": domain.ListMergeReplace},
			expected: `remote_servers:
  cluster:
    shard:
    - replica:
      - host: b
        port: 9000
`,
		},
		{
			name: "ScalarSet_Unioned",
			base: `service:
  pipelines:
    traces:
      receivers: [otlp]`,
			override: `service:
  pipelines:
    traces:
      receivers: [jaeger]`,
			strategy: map[string]domain.ListMerge{"service.pipelines.*.receivers": domain.ListMergeSet},
			expected: `service:
  pipelines:
    traces:
      receivers:
      - otlp
      - jaeger
`,
		},
		{
			name: "OrderedList_InsertsBeforeTerminal",
			base: `service:
  pipelines:
    traces:
      processors: [memory_limiter, batch]`,
			override: `service:
  pipelines:
    traces:
      processors: [resourcedetection]`,
			strategy: map[string]domain.ListMerge{"service.pipelines.*.processors": domain.ListMergeOrdered},
			expected: `service:
  pipelines:
    traces:
      processors:
      - memory_limiter
      - resourcedetection
      - batch
`,
		},
		{
			name: "MapListUnderSet_DegradesToAtomic",
			base: `raft_configuration:
  server:
  - hostname: a
    id: 0`,
			override: `raft_configuration:
  server:
  - hostname: b
    id: 0`,
			strategy: map[string]domain.ListMerge{"raft_configuration.server": domain.ListMergeSet},
			expected: `raft_configuration:
  server:
  - hostname: b
    id: 0
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.MergeYAML(tt.base, tt.override, tt.strategy)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
