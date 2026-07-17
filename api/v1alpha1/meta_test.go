package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeConfigSet_AllocatesAndOverwrites(t *testing.T) {
	var config TypeConfig // nil Data
	config.Set("collector/agent/agent.yaml", []byte("a: 1"))
	assert.Equal(t, "a: 1", config.Data["collector/agent/agent.yaml"])

	// Set on the same path overwrites in place.
	config.Set("collector/agent/agent.yaml", []byte("a: 2"))
	assert.Equal(t, "a: 2", config.Data["collector/agent/agent.yaml"])
}
