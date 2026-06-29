package telemetrykeepermolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryStore(t *testing.T) {
	t.Parallel()

	keeper, ok := keeperConfigTemplates.Resolve("25.5.6")
	assert.True(t, ok)
	assert.NotEmpty(t, keeper)

	// An unknown version falls back to the latest supported template.
	keeper, ok = keeperConfigTemplates.Resolve("0.0.0")
	assert.False(t, ok)
	assert.NotEmpty(t, keeper)
}
