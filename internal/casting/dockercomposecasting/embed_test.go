package dockercomposecasting

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotEmptyAndValid(t *testing.T) {
	assert.NotEmpty(t, composeYAMLTemplate)
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}
