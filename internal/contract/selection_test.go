package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectionFilter(t *testing.T) {
	substrate := MustNewSubstrate("foundry")

	tests := []struct {
		name          string
		selection     Selection
		expectedMatch map[TagKey]string
	}{
		{
			name:      "Substrate_MatchesEverythingItOwns",
			selection: substrate.Select(),
			expectedMatch: map[TagKey]string{
				TagKeyName: "foundry",
			},
		},
		{
			name:      "PrivateSubnet_MatchesTheType",
			selection: substrate.Select().WithSubnetType(SubnetTypePrivate),
			expectedMatch: map[TagKey]string{
				TagKeyName:       "foundry",
				TagKeySubnetType: "private",
			},
		},
		{
			name:      "PersistentClass_MatchesTheClass",
			selection: substrate.Select().WithStorage(StorageClassPersistent),
			expectedMatch: map[TagKey]string{
				TagKeyName:    "foundry",
				TagKeyStorage: "persistent",
			},
		},
		{
			name:      "EphemeralClass_MatchesTheClass",
			selection: substrate.Select().WithStorage(StorageClassEphemeral),
			expectedMatch: map[TagKey]string{
				TagKeyName:    "foundry",
				TagKeyStorage: "ephemeral",
			},
		},
		{
			name: "Claim_MatchesTheHolder",
			selection: substrate.Select().
				WithStorage(StorageClassPersistent).
				WithClaims(Identities{MustNewIdentity("telemetrystore", 0, 0)}),
			expectedMatch: map[TagKey]string{
				TagKeyName:       "foundry",
				TagKeyStorage:    "persistent",
				TagKeyIdentities: "telemetrystore-0-0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMatch, tt.selection.Match())
		})
	}
}
