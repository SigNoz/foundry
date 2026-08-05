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

// A consumer states a class and gets what the producer stamped on every node of
// it, at any ordinal. The class is the whole identity, so there is nothing else
// for the two sides to agree on.
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

// Claims are optional: a casting whose platform tracks the identity-to-disk
// binding itself never mentions them, and must then see no claim tag stamped and
// no claim key in its filter. Every casting but ECS is in that position, so this
// is the common path, not the edge.
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

// A resource's contract tags are not a parallel list that has to agree with the
// consumer's filter -- they are that filter. Asserted so the composition cannot
// be unwound back into two lists.
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

// A filter's keys are the only ones a consumer depends on the exact spelling of.
// Renaming one leaves live infrastructure unmatched, and that failure has no
// checkpoint: the filter returns nothing, after foundry has exited. Provenance
// keys carry no such constraint, which is why they are absent here.
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
