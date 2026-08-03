package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOwnership(t *testing.T) {
	tests := []struct {
		name              string
		out               string
		self              string
		expectedForeign   string
		expectedConflict  bool
		expectedUnlabeled bool
	}{
		{
			name: "Empty_NoOwnership",
			out:  "",
			self: "CollectionAgent",
		},
		{
			name: "SelfKind_NoConflict",
			out:  "CollectionAgent\n",
			self: "CollectionAgent",
		},
		{
			name:             "ForeignKind_Conflicts",
			out:              "Installation\n",
			self:             "CollectionAgent",
			expectedForeign:  "Installation",
			expectedConflict: true,
		},
		{
			name:              "UnlabeledOnly_UnlabeledWithoutConflict",
			out:               "\n\n",
			self:              "CollectionAgent",
			expectedUnlabeled: true,
		},
		{
			name:              "UnlabeledAndForeign_Conflicts",
			out:               "\nInstallation\n",
			self:              "CollectionAgent",
			expectedForeign:   "Installation",
			expectedConflict:  true,
			expectedUnlabeled: true,
		},
		{
			name:             "DuplicateForeign_SingleForeign",
			out:              "Installation\nInstallation\n",
			self:             "CollectionAgent",
			expectedForeign:  "Installation",
			expectedConflict: true,
		},
		{
			name:              "SelfAmongUnlabeled_NoConflict",
			out:               "CollectionAgent\n\nCollectionAgent\n",
			self:              "CollectionAgent",
			expectedUnlabeled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownership := ParseOwnership(tt.out)

			foreign, conflict := ownership.Foreign(tt.self)
			assert.Equal(t, tt.expectedForeign, foreign)
			assert.Equal(t, tt.expectedConflict, conflict)
			assert.Equal(t, tt.expectedUnlabeled, ownership.HasUnlabeled())
		})
	}
}
