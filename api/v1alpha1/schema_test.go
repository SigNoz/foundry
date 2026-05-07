package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustNewJSONSchema(t *testing.T) {
	assert.NotPanics(t, func() {
		mustNewJSONSchema()
	})
}
