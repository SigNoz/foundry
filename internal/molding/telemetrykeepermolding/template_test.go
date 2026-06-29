package telemetrykeepermolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryKeeperTemplates(t *testing.T) {
	tests := []struct {
		name    string
		version string
		exact   bool
	}{
		{"Version2556_ResolvesExact", "25.5.6", true},
		{"Version25125_ResolvesExact", "25.12.5", true},
		{"UnknownVersion_FallsBackToLatest", "0.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keeper, ok := keeperConfigTemplates.Resolve(tt.version)
			assert.Equal(t, tt.exact, ok)
			assert.NotEmpty(t, keeper)
		})
	}
}
