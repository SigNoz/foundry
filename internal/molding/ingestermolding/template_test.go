package ingestermolding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIngester(t *testing.T) {
	assert.NotEmpty(t, ConfigV0129xTemplate)
	assert.NotEmpty(t, OpampV0129xTemplate)
}
