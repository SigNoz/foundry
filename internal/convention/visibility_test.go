package convention

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Names are length-constrained and tag values are not, so visibility renders
// compact in a name and spelled out in a tag.
func TestVisibility(t *testing.T) {
	tests := []struct {
		name          string
		visibility    Visibility
		expectedWord  string
		expectedShort string
	}{
		{name: "Private_BothForms", visibility: VisibilityPrivate, expectedWord: "private", expectedShort: "prv"},
		{name: "Public_BothForms", visibility: VisibilityPublic, expectedWord: "public", expectedShort: "pub"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedWord, tt.visibility.String())
			assert.Equal(t, tt.expectedShort, tt.visibility.Short())
		})
	}
}

// The zero value renders nothing, which is how a resource with no network face
// drops the qualifier and the tag rather than carrying them empty.
func TestVisibilityZeroValueRendersNothing(t *testing.T) {
	assert.Empty(t, Visibility{}.String())
	assert.Empty(t, Visibility{}.Short())
}
