package dockercomposecasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/stretchr/testify/assert"
)

func TestNotEmptyAndValid(t *testing.T) {
	assert.NotEmpty(t, composeYAMLTemplate)

	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, collectionagent.Default())

	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}
