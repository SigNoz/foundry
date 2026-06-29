package telemetrykeepermolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryStore(t *testing.T) {
	t.Parallel()

	// Every supported version resolves to a dedicated, non-nil template.
	for _, version := range []string{"25.5.6", "25.12.5"} {
		keeper, ok := keeperConfigTemplates.Resolve(version)
		assert.True(t, ok, "keeper template should exist for %s", version)
		assert.NotEmpty(t, keeper)
	}

	// An unknown version falls back to the latest supported template.
	keeper, ok := keeperConfigTemplates.Resolve("0.0.0")
	assert.False(t, ok)
	assert.NotEmpty(t, keeper)
}
