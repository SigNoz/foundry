package collectormolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollector(t *testing.T) {
	assert.NotEmpty(t, agentConfig)
}
