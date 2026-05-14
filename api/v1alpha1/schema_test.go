package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustResolveSchema(t *testing.T) {
	t.Parallel()

	t.Run("valid schema", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, MustResolveSchema([]byte(`{"type": "object"}`)))
	})

	t.Run("invalid json panics", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			MustResolveSchema([]byte(`not json`))
		})
	})
}
