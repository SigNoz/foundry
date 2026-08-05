package convention

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestSelectionFilter(t *testing.T) {
	substrate := MustNewSubstrate("foundry")

	tests := []struct {
		name           string
		selection      Selection
		expectedFilter map[string]string
	}{
		{
			name:      "Substrate_MatchesEverythingItOwns",
			selection: substrate.Select(),
			expectedFilter: map[string]string{
				TagKeyName.String(): "foundry",
			},
		},
		{
			name:      "PersistentClass_MatchesTheClass",
			selection: substrate.Select().WithStorage(infrastructure.StorageClassPersistent),
			expectedFilter: map[string]string{
				TagKeyName.String():    "foundry",
				TagKeyStorage.String(): "persistent",
			},
		},
		{
			name:      "EphemeralClass_MatchesTheClass",
			selection: substrate.Select().WithStorage(infrastructure.StorageClassEphemeral),
			expectedFilter: map[string]string{
				TagKeyName.String():    "foundry",
				TagKeyStorage.String(): "ephemeral",
			},
		},
		{
			name: "Claim_MatchesTheHolder",
			selection: substrate.Select().
				WithStorage(infrastructure.StorageClassPersistent).
				WithClaims(Identities{MustNewIdentity("telemetrystore", 0, 0)}),
			expectedFilter: map[string]string{
				TagKeyName.String():       "foundry",
				TagKeyStorage.String():    "persistent",
				TagKeyIdentities.String(): "telemetrystore-0-0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedFilter, tt.selection.Filter())
		})
	}
}

// A consumer states a class; the producer stamps it per node. The filter has to
// match at any ordinal.
func TestClassFilterMatchesWhatTheProducerStamped(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	consumer := substrate.Select().WithStorage(infrastructure.StorageClassPersistent).Filter()

	for ordinal := range 3 {
		stamped := substrate.Node(infrastructure.StorageClassPersistent, ordinal).Tags()

		for key, value := range consumer {
			assert.Equal(t, value, stamped[key], "node %d does not match the class filter on %s", ordinal, key)
		}
	}
}

// A platform that tracks the identity-to-disk binding itself stamps no claim tag
// and filters on none.
func TestClaimsAreOptional(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	persistent := infrastructure.StorageClassPersistent

	volume := substrate.Volume(persistent, 0)
	assert.NotContains(t, volume.Tags(), TagKeyIdentities.String())
	assert.NotContains(t, volume.Filter(), TagKeyIdentities.String())

	selection := substrate.Select().WithStorage(infrastructure.StorageClassPersistent)
	assert.NotContains(t, selection.Filter(), TagKeyIdentities.String())

	// And an empty claim set is the same as never mentioning them.
	assert.Equal(t, volume.Tags(), substrate.Volume(persistent, 0).WithClaims(Identities{}).Tags())
}

// A resource's filter is a subset of its tags, not a parallel list that has to
// agree with them.
func TestResourceContractTagsAreItsSelection(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	zone := MustParseZone("us-east-1a")
	persistent := infrastructure.StorageClassPersistent

	resources := []Resource{
		substrate.Cluster(),
		substrate.VPC().WithKind(infrastructure.ResourceKindCollectionAgent),
		substrate.Subnet(VisibilityPrivate, zone),
		substrate.NATGateway(zone),
		substrate.Role(RoleExec),
		substrate.Node(persistent, 0),
		substrate.Volume(persistent, 1).WithClaims(Identities{MustNewIdentity("signoz", 0)}),
	}

	for _, resource := range resources {
		tags := resource.Tags()

		for key, value := range resource.Selection().Filter() {
			assert.Equal(t, value, tags[key], "the selection filters on %s differently to how it is stamped", key)
		}
	}
}

// A filter's keys are the only ones whose spelling live infrastructure depends
// on: renaming one leaves it unmatched, with no checkpoint to catch it.
func TestFilterKeysMatchDeployedSpelling(t *testing.T) {
	filter := MustNewSubstrate("foundry").Select().
		WithStorage(infrastructure.StorageClassPersistent).
		WithClaims(Identities{MustNewIdentity("signoz", 0)}).
		Filter()

	assert.Equal(t, map[string]string{
		"foundry.signoz.io/name":       "foundry",
		"foundry.signoz.io/storage":    "persistent",
		"foundry.signoz.io/identities": "signoz-0",
	}, filter)
}
