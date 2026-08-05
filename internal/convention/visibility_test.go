package convention

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A name is length-constrained where a tag value is not, so the two renderings
// differ.
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

// The zero value renders nothing, so a resource with no network face carries
// neither the qualifier nor the tag.
func TestVisibilityZeroValueRendersNothing(t *testing.T) {
	assert.Empty(t, Visibility{}.String())
	assert.Empty(t, Visibility{}.Short())
}
