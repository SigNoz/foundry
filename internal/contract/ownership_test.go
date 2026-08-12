package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOwnership(t *testing.T) {
	tests := []struct {
		name           string
		ownership      Ownership
		expectedWord   string
		expectedShared bool
	}{
		{name: "Owned_NotShared", ownership: OwnershipOwned, expectedWord: "owned", expectedShared: false},
		{name: "Shared_IsShared", ownership: OwnershipShared, expectedWord: "shared", expectedShared: true},

		// A caller says nothing when the substrate created the resource itself,
		// which is the common case, so the zero value has to mean owned in both
		// renderings rather than only in one.
		{name: "ZeroValue_OwnedInBothForms", ownership: Ownership{}, expectedWord: "owned", expectedShared: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedWord, tt.ownership.String())
			assert.Equal(t, tt.expectedShared, tt.ownership.IsShared())
		})
	}
}
