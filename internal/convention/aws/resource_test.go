package aws

import (
	"github.com/signoz/foundry/internal/convention"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestResourceName(t *testing.T) {
	substrate := convention.MustNewSubstrate("foundry")
	privateA := convention.MustNewKey("private-a")
	publicA := convention.MustNewKey("public-a")
	persistent := convention.NewNodeGroup(convention.MustNewKey("persistent"), v1alpha1.StorageClassPersistent)
	ephemeral := convention.NewNodeGroup(convention.MustNewKey("ephemeral"), v1alpha1.StorageClassEphemeral)

	tests := []struct {
		name         string
		resource     Resource
		expectedName string
	}{
		{name: "Cluster_Unqualified", resource: Cluster(substrate), expectedName: "foundry-cls"},
		{name: "VPC_Unqualified", resource: VPC(substrate), expectedName: "foundry-vpc"},
		{name: "InternetGateway_Unqualified", resource: InternetGateway(substrate), expectedName: "foundry-igw"},
		{name: "PrivateSubnet_Key", resource: Subnet(substrate, privateA, v1alpha1.SubnetTypePrivate), expectedName: "foundry-sub-private-a"},
		{name: "PublicSubnet_Key", resource: Subnet(substrate, publicA, v1alpha1.SubnetTypePublic), expectedName: "foundry-sub-public-a"},
		{name: "RouteTable_Key", resource: RouteTable(substrate, privateA), expectedName: "foundry-rt-private-a"},
		{name: "NATGateway_Key", resource: NATGateway(substrate, publicA), expectedName: "foundry-nat-public-a"},
		{name: "ElasticIP_Key", resource: ElasticIP(substrate, publicA), expectedName: "foundry-eip-public-a"},
		{name: "SecurityGroup_Role", resource: SecurityGroup(substrate, RoleTask), expectedName: "foundry-sg-task"},
		{name: "SecurityGroupRule_RoleAndPurpose", resource: SecurityGroupRule(substrate, RoleTask, convention.MustNewKey("intra-cluster")), expectedName: "foundry-sg-task-intra-cluster"},
		{name: "Role_Role", resource: IAMRole(substrate, RoleExec), expectedName: "foundry-iam-exec"},
		{name: "RolePolicy_RoleAndPurpose", resource: IAMRolePolicy(substrate, RoleTask, convention.MustNewKey("appconfig-read")), expectedName: "foundry-iam-task-appconfig-read"},
		{name: "InstanceProfile_Role", resource: InstanceProfile(substrate, RoleNode), expectedName: "foundry-prf-node"},
		{name: "LaunchTemplate_GroupKey", resource: LaunchTemplate(substrate, ephemeral), expectedName: "foundry-lt-ephemeral"},
		{name: "AutoscalingGroup_GroupKey", resource: AutoscalingGroup(substrate, ephemeral), expectedName: "foundry-asg-ephemeral"},
		{name: "Node_GroupKeyAndOrdinal", resource: Node(substrate, persistent, 0), expectedName: "foundry-node-persistent-0"},
		{name: "Volume_GroupKeyAndOrdinal", resource: Volume(substrate, persistent, 2), expectedName: "foundry-vol-persistent-2"},
		{name: "EphemeralNode_GroupKeyAndOrdinal", resource: Node(substrate, ephemeral, 1), expectedName: "foundry-node-ephemeral-1"},
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
	purpose := convention.MustNewKey("appconfig-read")
	maxRoleSuffix := len("-iam-exec-" + purpose.String())

	for _, name := range []string{"a", "foundry", "signoz-prod-eu-central"} {
		substrate := convention.MustNewSubstrate(name)
		assert.LessOrEqual(t, len(IAMRolePolicy(substrate, RoleExec, purpose).Name())-len(name), maxRoleSuffix)
	}
}

// Adopting a resource must not rename it: the name belongs to whoever created it.
func TestSharedResourceKeepsItsName(t *testing.T) {
	substrate := convention.MustNewSubstrate("foundry")
	shared := VPC(substrate).WithOwnership(convention.OwnershipShared)

	assert.Equal(t, VPC(substrate).Name(), shared.Name())
	assert.NotContains(t, shared.Tags(), displayName)
	assert.Equal(t, "shared", shared.Tags()[Tag(convention.TagKeyOwner)])
}

func TestResourceTags(t *testing.T) {
	substrate := convention.MustNewSubstrate("foundry")
	privateA := convention.MustNewKey("private-a")
	persistent := convention.NewNodeGroup(convention.MustNewKey("persistent"), v1alpha1.StorageClassPersistent)

	tests := []struct {
		name            string
		resource        Resource
		expectedPresent map[string]string
		expectedAbsent  []convention.TagKey
	}{
		{
			name:     "Cluster_CarriesIdentityAndOwner",
			resource: Cluster(substrate),
			expectedPresent: map[string]string{
				Tag(convention.TagKeyName):  "foundry",
				Tag(convention.TagKeyOwner): "owned",
				displayName:                 "foundry-cls",
			},
			expectedAbsent: []convention.TagKey{convention.TagKeySubnetType, convention.TagKeyStorage, convention.TagKeyIdentities},
		},
		{
			name:     "PrivateSubnet_CarriesItsTypeSpelledOut",
			resource: Subnet(substrate, privateA, v1alpha1.SubnetTypePrivate),
			expectedPresent: map[string]string{
				displayName:                      "foundry-sub-private-a",
				Tag(convention.TagKeySubnetType): "private",
			},
			expectedAbsent: []convention.TagKey{convention.TagKeyStorage},
		},
		{
			name:     "PersistentNode_CarriesStorageFromItsGroup",
			resource: Node(substrate, persistent, 0),
			expectedPresent: map[string]string{
				displayName:                   "foundry-node-persistent-0",
				Tag(convention.TagKeyStorage): "persistent",
			},
			expectedAbsent: []convention.TagKey{convention.TagKeySubnetType, convention.TagKeyIdentities},
		},
		{
			name: "ClaimedVolume_CarriesIdentities",
			resource: Volume(substrate, persistent, 0).WithClaims(convention.Identities{
				convention.MustNewIdentity("telemetrystore", 0, 0),
				convention.MustNewIdentity("metastore", 0),
			}),
			expectedPresent: map[string]string{
				displayName:                      "foundry-vol-persistent-0",
				Tag(convention.TagKeyStorage):    "persistent",
				Tag(convention.TagKeyIdentities): "metastore-0,telemetrystore-0-0",
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
				assert.NotContains(t, tags, Tag(key))
			}
		})
	}
}

func TestResourceFilter(t *testing.T) {
	substrate := convention.MustNewSubstrate("foundry")
	privateA := convention.MustNewKey("private-a")
	persistent := convention.NewNodeGroup(convention.MustNewKey("persistent"), v1alpha1.StorageClassPersistent)

	tests := []struct {
		name           string
		resource       Resource
		expectedFilter map[string]string
	}{
		{
			name:     "VPC_SelectsIdentityOnly",
			resource: VPC(substrate),
			expectedFilter: map[string]string{
				Tag(convention.TagKeyName): "foundry",
			},
		},
		{
			name:     "ProvenanceOnly_IsNotSelectedOn",
			resource: Cluster(substrate),
			expectedFilter: map[string]string{
				Tag(convention.TagKeyName): "foundry",
			},
		},
		{
			name:     "PrivateSubnet_SelectsIdentityAndType",
			resource: Subnet(substrate, privateA, v1alpha1.SubnetTypePrivate),
			expectedFilter: map[string]string{
				Tag(convention.TagKeyName):       "foundry",
				Tag(convention.TagKeySubnetType): "private",
			},
		},
		{
			name:     "PersistentNode_SelectsIdentityAndStorage",
			resource: Node(substrate, persistent, 0),
			expectedFilter: map[string]string{
				Tag(convention.TagKeyName):    "foundry",
				Tag(convention.TagKeyStorage): "persistent",
			},
		},
		{
			name:     "ClaimedVolume_SelectsTheClaim",
			resource: Volume(substrate, persistent, 0).WithClaims(convention.Identities{convention.MustNewIdentity("signoz", 0)}),
			expectedFilter: map[string]string{
				Tag(convention.TagKeyName):       "foundry",
				Tag(convention.TagKeyStorage):    "persistent",
				Tag(convention.TagKeyIdentities): "signoz-0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedFilter, tt.resource.Filter())
		})
	}
}

// A fact stated once must render the same way wherever it is read back. The
// operator's key reaches the name; the closed enums reach the tags, which is
// all a consuming casting can predict.
func TestNameAndTagsAgreeOnTheSameFact(t *testing.T) {
	substrate := convention.MustNewSubstrate("foundry")

	for _, subnetType := range []v1alpha1.SubnetType{v1alpha1.SubnetTypePrivate, v1alpha1.SubnetTypePublic} {
		key := convention.MustNewKey(subnetType.String() + "-a")
		subnet := Subnet(substrate, key, subnetType)

		assert.Contains(t, subnet.Name(), key.String())
		assert.Equal(t, subnetType.String(), subnet.Tags()[Tag(convention.TagKeySubnetType)])
	}

	for _, storage := range []v1alpha1.StorageClass{v1alpha1.StorageClassPersistent, v1alpha1.StorageClassEphemeral} {
		group := convention.NewNodeGroup(convention.MustNewKey(storage.String()), storage)
		node := Node(substrate, group, 0)

		assert.Contains(t, node.Name(), group.Key().String())
		assert.Equal(t, storage.String(), node.Tags()[Tag(convention.TagKeyStorage)])
	}
}

// Two types sharing a short form would derive the same name shape. A security
// group and a role each cover two constructors, distinguished by the trailing
// purpose rather than by a second short form.
func TestResourceTypeShortFormsAreDistinct(t *testing.T) {
	resourceTypes := []resourceType{
		typeCluster, typeVPC, typeInternetGateway, typeSubnet, typeRouteTable,
		typeNATGateway, typeElasticIP, typeSecurityGroup, typeRole,
		typeInstanceProfile, typeLaunchTemplate, typeAutoscalingGroup,
		typeNode, typeVolume,
	}

	seen := make(map[string]struct{}, len(resourceTypes))
	for _, resource := range resourceTypes {
		assert.NotContains(t, seen, resource.String())
		seen[resource.String()] = struct{}{}
	}
}

// An empty qualifier drops its segment, so one declaration serves both a
// security group and the rules attached to it.
func TestUnsetQualifierDropsFromTheName(t *testing.T) {
	substrate := convention.MustNewSubstrate("foundry")

	assert.Equal(t, "foundry-sg-task", SecurityGroup(substrate, RoleTask).Name())
	assert.Equal(t, "foundry-sg-task-intra-cluster", SecurityGroupRule(substrate, RoleTask, convention.MustNewKey("intra-cluster")).Name())
}

// A declared qualifier that renders nothing would silently drop a segment meant
// to distinguish the name.
func TestEveryDeclaredQualifierContributes(t *testing.T) {
	substrate := convention.MustNewSubstrate("foundry")
	privateA := convention.MustNewKey("private-a")
	persistent := convention.NewNodeGroup(convention.MustNewKey("persistent"), v1alpha1.StorageClassPersistent)

	tests := []struct {
		name             string
		resource         Resource
		expectedSegments int
	}{
		{name: "VPC_NoQualifier", resource: VPC(substrate), expectedSegments: 0},
		{name: "Subnet_Key", resource: Subnet(substrate, privateA, v1alpha1.SubnetTypePrivate), expectedSegments: 1},
		{name: "NATGateway_Key", resource: NATGateway(substrate, privateA), expectedSegments: 1},
		{name: "Role_Role", resource: IAMRole(substrate, RoleExec), expectedSegments: 1},
		{name: "RolePolicy_RoleAndPurpose", resource: IAMRolePolicy(substrate, RoleExec, convention.MustNewKey("ssm-session")), expectedSegments: 2},
		{name: "InstanceProfile_Role", resource: InstanceProfile(substrate, RoleNode), expectedSegments: 1},
		{name: "LaunchTemplate_GroupKey", resource: LaunchTemplate(substrate, persistent), expectedSegments: 1},
		{name: "Node_GroupKeyAndOrdinal", resource: Node(substrate, persistent, 0), expectedSegments: 2},
		{name: "Volume_GroupKeyAndOrdinal", resource: Volume(substrate, persistent, 0), expectedSegments: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := 0
			for _, qualifier := range tt.resource.resourceType.qualifiers {
				if qualifier.of(tt.resource) != "" {
					rendered++
				}
			}

			assert.Equal(t, tt.expectedSegments, rendered)
		})
	}
}

// Ordinal zero is a real ordinal, so only types that have one declare it.
func TestOrdinalZeroRenders(t *testing.T) {
	substrate := convention.MustNewSubstrate("foundry")
	persistent := convention.NewNodeGroup(convention.MustNewKey("persistent"), v1alpha1.StorageClassPersistent)

	assert.Equal(t, "foundry-node-persistent-0", Node(substrate, persistent, 0).Name())
	assert.Equal(t, "foundry-vpc", VPC(substrate).Name())
}
