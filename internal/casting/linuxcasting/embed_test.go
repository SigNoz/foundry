package linuxcasting

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotEmptyAndValid(t *testing.T) {
	assert.NotEmpty(t, signozServiceTemplate)
	assert.NotEmpty(t, ingesterServiceTemplate)
	buf := bytes.NewBuffer(nil)
	err := signozServiceTemplate.Execute(buf, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}
