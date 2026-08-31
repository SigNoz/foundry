package systemdbinarycasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/stretchr/testify/assert"
)

func TestNotEmptyAndValid(t *testing.T) {
	assert.NotEmpty(t, serviceTemplate)

	buf := bytes.NewBuffer(nil)
	err := serviceTemplate.Execute(buf, collectionagent.Default())

	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())

	assert.NotEmpty(t, agentYAMLTemplate)

	buf.Reset()
	err = agentYAMLTemplate.Execute(buf, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}
