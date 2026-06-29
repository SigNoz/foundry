package telemetrystoremolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryStore(t *testing.T) {
	t.Parallel()

	// Every supported version resolves to a dedicated, non-nil template.
	for _, version := range []string{"25.5.6", "25.12.5"} {
		config, ok := clickhouseConfigTemplates.Resolve(version)
		assert.True(t, ok, "config template should exist for %s", version)
		assert.NotEmpty(t, config)

		functions, ok := clickhouseFunctionsTemplates.Resolve(version)
		assert.True(t, ok, "functions template should exist for %s", version)
		assert.NotEmpty(t, functions)
	}

	// An unknown version falls back to the latest supported template.
	config, ok := clickhouseConfigTemplates.Resolve("0.0.0")
	assert.False(t, ok)
	assert.NotEmpty(t, config)
}
