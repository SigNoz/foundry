package convention

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestResourceName(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	zone := MustParseZone("us-east-1a")
	persistent := infrastructure.StorageClassPersistent
	ephemeral := infrastructure.StorageClassEphemeral

	tests := []struct {
		name         string
		resource     Resource
		expectedName string
	}{
		{name: "Cluster_Unqualified", resource: substrate.Cluster(), expectedName: "foundry-cls"},
		{name: "VPC_Unqualified", resource: substrate.VPC(), expectedName: "foundry-vpc"},
		{name: "InternetGateway_Unqualified", resource: substrate.InternetGateway(), expectedName: "foundry-igw"},
		{name: "PrivateSubnet_VisibilityAndZone", resource: substrate.Subnet(VisibilityPrivate, zone), expectedName: "foundry-sub-prv-east1a"},
		{name: "PublicSubnet_VisibilityAndZone", resource: substrate.Subnet(VisibilityPublic, zone), expectedName: "foundry-sub-pub-east1a"},
		{name: "PrivateRouteTable_PerZone", resource: substrate.RouteTableInZone(VisibilityPrivate, zone), expectedName: "foundry-rt-prv-east1a"},
		{name: "PublicRouteTable_ZoneShared", resource: substrate.RouteTable(VisibilityPublic), expectedName: "foundry-rt-pub"},
		{name: "NATGateway_PerZone", resource: substrate.NATGateway(zone), expectedName: "foundry-nat-east1a"},
		{name: "TaskSecurityGroup_Role", resource: substrate.SecurityGroup(RoleTask), expectedName: "foundry-sg-task"},
		{name: "ExecRole_Role", resource: substrate.Role(RoleExec), expectedName: "foundry-iam-exec"},
		{name: "Node_ClassAndOrdinal", resource: substrate.Node(persistent, 0), expectedName: "foundry-node-persistent-0"},
		{name: "Volume_ClassAndOrdinal", resource: substrate.Volume(persistent, 2), expectedName: "foundry-vol-persistent-2"},
		{name: "EphemeralNode_ClassAndOrdinal", resource: substrate.Node(ephemeral, 1), expectedName: "foundry-node-ephemeral-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, tt.resource.Name())
		})
	}
}

// A role name is the longest suffix a caller has to budget for against its
// provider's cap, which this package does not know.
func TestRoleNameOverheadIsBounded(t *testing.T) {
	const maxRoleSuffix = len("-iam-exec")

	for _, name := range []string{"a", "foundry", "signoz-prod-eu-central"} {
		substrate := MustNewSubstrate(name)
		assert.LessOrEqual(t, len(substrate.Role(RoleExec).Name())-len(name), maxRoleSuffix)
	}
}

// Adopting a resource must not rename it: the name belongs to whoever created it.
func TestSharedResourceKeepsItsName(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	shared := substrate.VPC().WithOwnership(OwnershipShared)

	assert.Equal(t, substrate.VPC().Name(), shared.Name())
	assert.NotContains(t, shared.Tags(), TagKeyDisplayName.String())
	assert.Equal(t, "shared", shared.Tags()[TagKeyOwner.String()])
}

func TestResourceTags(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	zone := MustParseZone("us-east-1a")
	persistent := infrastructure.StorageClassPersistent

	tests := []struct {
		name            string
		resource        Resource
		expectedPresent map[string]string
		expectedAbsent  []TagKey
	}{
		{
			name:     "Cluster_CarriesIdentityAndOwner",
			resource: substrate.Cluster(),
			expectedPresent: map[string]string{
				TagKeyName.String():        "foundry",
				TagKeyOwner.String():       "owned",
				TagKeyDisplayName.String(): "foundry-cls",
			},
			expectedAbsent: []TagKey{TagKeyVisibility, TagKeyStorage, TagKeyIdentities, TagKeyResourceKind},
		},
		{
			name:     "PrivateSubnet_CarriesVisibilitySpelledOut",
			resource: substrate.Subnet(VisibilityPrivate, zone).WithKind(infrastructure.ResourceKindInstallation),
			expectedPresent: map[string]string{
				TagKeyDisplayName.String():  "foundry-sub-prv-east1a",
				TagKeyVisibility.String():   "private",
				TagKeyResourceKind.String(): "Installation",
			},
			expectedAbsent: []TagKey{TagKeyStorage},
		},
		{
			name:     "PersistentNode_CarriesStorageFromItsGroup",
			resource: substrate.Node(persistent, 0),
			expectedPresent: map[string]string{
				TagKeyDisplayName.String(): "foundry-node-persistent-0",
				TagKeyStorage.String():     "persistent",
			},
			expectedAbsent: []TagKey{TagKeyVisibility, TagKeyIdentities},
		},
		{
			name: "ClaimedVolume_CarriesIdentities",
			resource: substrate.Volume(persistent, 0).WithClaims(Identities{
				MustNewIdentity("telemetrystore", 0, 0),
				MustNewIdentity("metastore", 0),
			}),
			expectedPresent: map[string]string{
				TagKeyDisplayName.String(): "foundry-vol-persistent-0",
				TagKeyStorage.String():     "persistent",
				TagKeyIdentities.String():  "metastore-0,telemetrystore-0-0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := tt.resource.Tags()

			for key, expected := range tt.expectedPresent {
				assert.Equal(t, expected, tags[key], "tag %s", key)
			}

			for _, key := range tt.expectedAbsent {
				assert.NotContains(t, tags, key.String())
			}
		})
	}
}

func TestResourceFilter(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	zone := MustParseZone("us-east-1a")
	persistent := infrastructure.StorageClassPersistent

	tests := []struct {
		name           string
		resource       Resource
		expectedFilter map[string]string
	}{
		{
			name:     "VPC_SelectsIdentityOnly",
			resource: substrate.VPC(),
			expectedFilter: map[string]string{
				TagKeyName.String(): "foundry",
			},
		},
		{
			name:     "ProvenanceOnly_IsNotSelectedOn",
			resource: substrate.Subnet(VisibilityPrivate, zone).WithKind(infrastructure.ResourceKindInstallation),
			expectedFilter: map[string]string{
				TagKeyName.String(): "foundry",
			},
		},
		{
			name:     "PersistentNode_SelectsIdentityAndStorage",
			resource: substrate.Node(persistent, 0),
			expectedFilter: map[string]string{
				TagKeyName.String():    "foundry",
				TagKeyStorage.String(): "persistent",
			},
		},
		{
			name:     "ClaimedVolume_SelectsTheClaim",
			resource: substrate.Volume(persistent, 0).WithClaims(Identities{MustNewIdentity("signoz", 0)}),
			expectedFilter: map[string]string{
				TagKeyName.String():       "foundry",
				TagKeyStorage.String():    "persistent",
				TagKeyIdentities.String(): "signoz-0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedFilter, tt.resource.Filter())
		})
	}
}

// A fact stated once must render the same way in the name and in the tag.
func TestNameAndTagsAgreeOnTheSameFact(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	zone := MustParseZone("us-east-1a")

	for _, visibility := range []Visibility{VisibilityPrivate, VisibilityPublic} {
		subnet := substrate.Subnet(visibility, zone)

		assert.Contains(t, subnet.Name(), visibility.Short())
		assert.Equal(t, visibility.String(), subnet.Tags()[TagKeyVisibility.String()])
	}

	for _, storage := range []infrastructure.StorageClass{infrastructure.StorageClassPersistent, infrastructure.StorageClassEphemeral} {
		node := substrate.Node(storage, 0)

		assert.Contains(t, node.Name(), storage.String())
		assert.Equal(t, storage.String(), node.Tags()[TagKeyStorage.String()])
	}
}
