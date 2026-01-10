package dockercomposecasting

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotEmptyAndValid(t *testing.T) {
	assert.NotEmpty(t, ComposeYAMLTemplate)
	buf := bytes.NewBuffer(nil)
	err := ComposeYAMLTemplate.Execute(buf, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}
