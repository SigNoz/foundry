package yamls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYAMLs(t *testing.T) {
	assert.NotEmpty(t, ConfigIngesterV0129xYAML)
	assert.NotEmpty(t, ConfigClickhousev2556YAML)
	assert.NotEmpty(t, FunctionsClickhousev2556YAML)
	assert.NotEmpty(t, KeeperClickhousev2556YAML)
}
