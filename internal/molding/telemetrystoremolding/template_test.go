package telemetrystoremolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryStore(t *testing.T) {
	t.Parallel()

	config, ok := clickhouseConfigTemplates.Resolve("25.5.6")
	assert.True(t, ok)
	assert.NotEmpty(t, config)

	functions, ok := clickhouseFunctionsTemplates.Resolve("25.5.6")
	assert.True(t, ok)
	assert.NotEmpty(t, functions)

	// An unknown version falls back to the latest supported template.
	config, ok = clickhouseConfigTemplates.Resolve("0.0.0")
	assert.False(t, ok)
	assert.NotEmpty(t, config)
}
