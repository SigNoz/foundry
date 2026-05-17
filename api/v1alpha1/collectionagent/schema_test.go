package collectionagent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, Schema())
}

func TestSchemaValidatesDefault(t *testing.T) {
	t.Parallel()

	contents, err := json.Marshal(Default())
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(contents, &payload))

	assert.NoError(t, Schema().Validate(payload))
}
