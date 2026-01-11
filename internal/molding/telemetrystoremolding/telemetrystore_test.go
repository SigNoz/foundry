package telemetrystoremolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryStore(t *testing.T) {
	assert.NotEmpty(t, ConfigClickhousev2556YAML)
	assert.NotEmpty(t, FunctionsClickhousev2556YAML)
	assert.NotEmpty(t, KeeperClickhousev2556YAML)
}
