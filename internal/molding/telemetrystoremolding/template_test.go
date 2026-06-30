package telemetrystoremolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryStore(t *testing.T) {
	assert.NotEmpty(t, ConfigClickhousev25125YAML)
	assert.NotEmpty(t, FunctionsClickhousev25125YAML)
}
