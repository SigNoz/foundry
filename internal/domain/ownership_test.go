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
