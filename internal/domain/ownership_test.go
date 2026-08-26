package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOwnerIsZero(t *testing.T) {
	tests := []struct {
		name         string
		owner        Owner
		expectedZero bool
	}{
		{name: "Nil_Zero", owner: nil, expectedZero: true},
		{name: "Empty_Zero", owner: Owner{}, expectedZero: true},
		{name: "AllValuesEmpty_Zero", owner: Owner{"kind": "", "name": ""}, expectedZero: true},
		{name: "OneValueSet_NotZero", owner: Owner{"kind": "", "name": "signoz"}, expectedZero: false},
		{name: "AllValuesSet_NotZero", owner: Owner{"kind": "Installation", "name": "signoz"}, expectedZero: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedZero, tt.owner.IsZero())
		})
	}
}

func TestOwnerRead(t *testing.T) {
	tests := []struct {
		name          string
		owner         Owner
		recorded      map[string]string
		expectedOwner Owner
	}{
		{
			name:          "AskedKeysOnly_Kept",
			owner:         Owner{"kind": "", "name": ""},
			recorded:      map[string]string{"kind": "Installation", "name": "signoz", "owner": "helm"},
			expectedOwner: Owner{"kind": "Installation", "name": "signoz"},
		},
		{
			name:          "MissingKey_Empty",
			owner:         Owner{"kind": "", "name": ""},
			recorded:      map[string]string{"kind": "Installation"},
			expectedOwner: Owner{"kind": "Installation", "name": ""},
		},
		{
			name:          "NothingAsked_Empty",
			owner:         Owner{},
			recorded:      map[string]string{"kind": "Installation"},
			expectedOwner: Owner{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedOwner, tt.owner.Read(tt.recorded))
		})
	}
}

func TestOwnerEqual(t *testing.T) {
	tests := []struct {
		name          string
		owner         Owner
		other         Owner
		expectedEqual bool
	}{
		{
			name:          "SameAttributes_Equal",
			owner:         Owner{"kind": "Installation", "name": "signoz"},
			other:         Owner{"kind": "Installation", "name": "signoz"},
			expectedEqual: true,
		},
		{
			// One attribute of the set differing is a different owner: an
			// owner is compared as a whole.
			name:          "OneAttributeDiffers_NotEqual",
			owner:         Owner{"kind": "Installation", "name": "signoz"},
			other:         Owner{"kind": "CollectionAgent", "name": "signoz"},
			expectedEqual: false,
		},
		{
			name:          "AbsentMatchesEmpty_Equal",
			owner:         Owner{"kind": "Installation", "name": ""},
			other:         Owner{"kind": "Installation"},
			expectedEqual: true,
		},
		{
			name:          "ExtraAttribute_NotEqual",
			owner:         Owner{"kind": "Installation"},
			other:         Owner{"kind": "Installation", "name": "signoz"},
			expectedEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedEqual, tt.owner.Equal(tt.other))
			assert.Equal(t, tt.expectedEqual, tt.other.Equal(tt.owner))
		})
	}
}

func TestOwnerString(t *testing.T) {
	owner := Owner{"name": "signoz", "kind": "Installation", "managed-by": "foundry"}

	assert.Equal(t, "kind=Installation,managed-by=foundry,name=signoz", owner.String())
	assert.Empty(t, Owner{}.String())
}

func TestOwnership(t *testing.T) {
	installation := Owner{"kind": "Installation", "name": "signoz"}
	agent := Owner{"kind": "CollectionAgent", "name": "signoz"}

	tests := []struct {
		name            string
		owners          []Owner
		self            Owner
		expectedForeign Owner
		expectedUnowned bool
	}{
		{
			name:   "NoWorkloads_NothingOwned",
			owners: nil,
			self:   installation,
		},
		{
			name:   "OnlySelf_NoConflict",
			owners: []Owner{installation, installation},
			self:   installation,
		},
		{
			name:            "OtherOwner_Conflict",
			owners:          []Owner{agent},
			self:            installation,
			expectedForeign: agent,
		},
		{
			name:            "SelfBesideOther_Conflict",
			owners:          []Owner{installation, agent},
			self:            installation,
			expectedForeign: agent,
		},
		{
			name:            "NothingRecorded_Unowned",
			owners:          []Owner{{"kind": "", "name": ""}},
			self:            installation,
			expectedUnowned: true,
		},
		{
			name:            "SelfBesideUnrecorded_UnownedNoConflict",
			owners:          []Owner{installation, {"kind": "", "name": ""}},
			self:            installation,
			expectedUnowned: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownership := NewOwnership(tt.owners...)

			foreign, conflict := ownership.Foreign(tt.self)
			assert.Equal(t, tt.expectedForeign != nil, conflict)
			assert.Equal(t, tt.expectedForeign, foreign)
			assert.Equal(t, tt.expectedUnowned, ownership.HasUnowned())
		})
	}
}

// The same owner reported by many workloads is one owner.
func TestOwnershipDeduplicates(t *testing.T) {
	installation := Owner{"kind": "Installation", "name": "signoz"}

	ownership := NewOwnership(installation, installation, installation)

	assert.Len(t, ownership.owners, 1)
}

// ParseOwner is String's inverse, so an owner survives a round trip through its
// own string form.
func TestParseOwner(t *testing.T) {
	tests := []struct {
		name  string
		owner Owner
	}{
		{name: "Attributes_RoundTrip", owner: Owner{"kind": "Installation", "name": "signoz"}},
		{name: "EmptyValues_RoundTrip", owner: Owner{"kind": "", "name": ""}},
		{name: "None_RoundTrip", owner: Owner{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.owner, ParseOwner(tt.owner.String()))
		})
	}
}

// ParseOwnership reads one owner per line, each in Owner.String form, so a line
// with no attributes marks the group partly unowned rather than becoming an
// owner whose every value is empty.
func TestParseOwnership(t *testing.T) {
	t.Run("Rows_NoConflict", func(t *testing.T) {
		ownership := ParseOwnership("kind=Installation,name=signoz\nkind=Installation,name=signoz\n")

		_, conflict := ownership.Foreign(Owner{"kind": "Installation", "name": "signoz"})

		assert.False(t, conflict)
		assert.False(t, ownership.HasUnowned())
	})

	t.Run("ForeignRow_Conflict", func(t *testing.T) {
		ownership := ParseOwnership("kind=Installation,name=other\n")

		_, conflict := ownership.Foreign(Owner{"kind": "Installation", "name": "signoz"})

		assert.True(t, conflict)
	})

	t.Run("EmptyValues_Unowned", func(t *testing.T) {
		ownership := ParseOwnership("kind=,name=\n")

		assert.True(t, ownership.HasUnowned())
	})

	t.Run("NoOutput_Empty", func(t *testing.T) {
		ownership := ParseOwnership("")

		assert.False(t, ownership.HasUnowned())
	})
}
