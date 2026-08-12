package eks

import (
	"maps"
	"slices"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/contract"
	"github.com/signoz/foundry/internal/contract/aws"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const oneZone = `networking:
  networkCIDR: 10.0.0.0/16
  subnets:
    private-a: {type: private, zone: us-east-1a, cidr: 10.0.0.0/19}
    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}
instanceGroups:
  persistent: {storage: persistent, machineType: m5.large, minSize: 3, maxSize: 3}
  ephemeral: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1}
`

const twoZones = `networking:
  networkCIDR: 10.0.0.0/16
  subnets:
    private-a: {type: private, zone: us-east-1a, cidr: 10.0.0.0/19}
    private-b: {type: private, zone: us-east-1b, cidr: 10.0.32.0/19}
    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}
    public-b: {type: public, zone: us-east-1b, cidr: 10.0.100.0/22}
instanceGroups:
  persistent: {storage: persistent, machineType: m5.large, minSize: 3, maxSize: 3}
  ephemeral: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1}
`

// deriveData renders a declaration written the way the molding settles it.
func deriveData(t *testing.T, declaration string) (*Resources, error) {
	t.Helper()

	config := &infrastructure.ResourceConfig{}
	require.NoError(t, domain.UnmarshalYAML([]byte(declaration), config))

	return Derive(contract.MustNewSubstrate("foundry"), config, nil)
}

func mustDeriveData(t *testing.T, declaration string) *Resources {
	t.Helper()

	derived, err := deriveData(t, declaration)
	require.NoError(t, err)

	return derived
}

// These names reach live infrastructure. Changing one replaces the resource it
// belongs to rather than updating it.
func TestTemplateData(t *testing.T) {
	derived := mustDeriveData(t, twoZones)

	tests := []struct {
		name         string
		of           func(*Resources) string
		expectedName string
	}{
		{name: "Cluster_Unqualified", of: func(d *Resources) string { return d.Cluster.Name }, expectedName: "foundry-cls"},
		{name: "VPC_Unqualified", of: func(d *Resources) string { return d.Network.VPC.Name }, expectedName: "foundry-vpc"},
		{name: "InternetGateway_Unqualified", of: func(d *Resources) string { return d.Network.InternetGateway.Name }, expectedName: "foundry-igw"},
		{name: "Subnet_Keyed", of: func(d *Resources) string { return d.Network.Subnets["private-a"].Name }, expectedName: "foundry-sub-private-a"},
		{name: "NATGateway_KeyedByTheSubnetItServes", of: func(d *Resources) string { return d.Network.NATGateways["private-a"].Name }, expectedName: "foundry-nat-private-a"},
		{name: "Role_ControlPlane", of: func(d *Resources) string { return d.Roles["cluster"].Name }, expectedName: "foundry-iam-cluster"},
		{name: "Role_Node", of: func(d *Resources) string { return d.Roles["node"].Name }, expectedName: "foundry-iam-node"},
		{name: "Role_StorageDriver", of: func(d *Resources) string { return d.Roles["ebs-csi"].Name }, expectedName: "foundry-iam-ebs-csi"},
		{name: "NodeGroup_GroupKey", of: func(d *Resources) string {
			return d.Groups["persistent"].NodeGroup.Name
		}, expectedName: "foundry-ng-persistent"},
		{name: "NodeGroup_ScalingGroupKey", of: func(d *Resources) string {
			return d.Groups["ephemeral"].NodeGroup.Name
		}, expectedName: "foundry-ng-ephemeral"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, tt.of(derived))
		})
	}
}

// The provider owns every node, including a stateful group's. The data has no
// field for a node, a volume, a launch template or an autoscaling group, so
// what remains to check is that every group resolves to a named pool.
func TestTemplateDataNamesOnlyThePool(t *testing.T) {
	derived := mustDeriveData(t, twoZones)

	for key, group := range derived.Groups {
		assert.NotEmpty(t, group.NodeGroup.Name, "group %s names no pool", key)
	}
}

// The substrate's roles are its own: control plane, node, and storage driver.
// A tenant workload's identity is not derived here; an instance profile or a
// security group of the substrate's making has no field to exist in.
func TestTemplateDataLeavesWorkloadIdentityAlone(t *testing.T) {
	derived := mustDeriveData(t, twoZones)

	assert.Equal(t, []string{"cluster", "ebs-csi", "node"}, slices.Sorted(maps.Keys(derived.Roles)))
}

// The tags a consuming casting filters on are the only ones whose spelling live
// infrastructure depends on.
func TestTemplateDataContractTags(t *testing.T) {
	derived := mustDeriveData(t, twoZones)

	assert.Equal(t, "private", derived.Network.Subnets["private-a"].Tags[aws.Tag(contract.TagKeySubnetType)])
	assert.Equal(t, "public", derived.Network.Subnets["public-a"].Tags[aws.Tag(contract.TagKeySubnetType)])

	assert.Equal(t, map[string]string{
		aws.Tag(contract.TagKeyName):    "foundry",
		aws.Tag(contract.TagKeyStorage): "persistent",
	}, derived.Groups["persistent"].Selector)

	// The control plane stamps its own ownership tag on what it discovers, so a
	// casting that reconciles has to be told to leave it alone.
	assert.Equal(t, []string{tagClusterPrefix + "foundry-cls"}, derived.IgnoredTags)
}

// A load balancer controller picks its subnets by these, and puts an
// internet-facing balancer in a public subnet and an internal one in a private.
func TestTemplateDataElectsSubnetsForLoadBalancers(t *testing.T) {
	derived := mustDeriveData(t, twoZones)

	assert.Equal(t, tagRoleValue, derived.Network.Subnets["public-a"].Tags[tagRoleELB])
	assert.NotContains(t, derived.Network.Subnets["public-a"].Tags, tagRoleInternalELB)

	assert.Equal(t, tagRoleValue, derived.Network.Subnets["private-a"].Tags[tagRoleInternalELB])
	assert.NotContains(t, derived.Network.Subnets["private-a"].Tags, tagRoleELB)
}

// An adopted subnet is not the substrate's to tag, so it carries whatever its
// owner already put on it.
func TestTemplateDataAdoptedSubnetsAreNotTagged(t *testing.T) {
	derived := mustDeriveData(t, `networking:
  networkID: vpc-0a1b2c3d
  subnets:
    private-a: {type: private, zone: us-east-1a, id: subnet-0a1b2c3d}
    private-b: {type: private, zone: us-east-1b, id: subnet-1b2c3d4e}
    public-a: {type: public, zone: us-east-1a, id: subnet-4e5f6a7b}
instanceGroups:
  ephemeral: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1}
`)

	assert.Empty(t, derived.Network.Subnets["private-a"].Tags)
	assert.Empty(t, derived.Network.Subnets["public-a"].Tags)
	assert.Equal(t, "vpc-0a1b2c3d", derived.Network.VPC.ID)
	assert.Empty(t, derived.Network.VPC.Name)
}

// A group placed nowhere would be a pool the provider cannot start a node in.
func TestTemplateDataWithoutAPrivateSubnet(t *testing.T) {
	_, err := deriveData(t, `networking:
  networkCIDR: 10.0.0.0/16
  subnets:
    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}
    public-b: {type: public, zone: us-east-1b, cidr: 10.0.100.0/22}
instanceGroups:
  ephemeral: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1}
`)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "there is no private subnet to place it in")
}

// The control plane's own interfaces are spread by the provider, which refuses
// a cluster it cannot spread. Failing here beats failing on apply.
func TestTemplateDataWithinOneZone(t *testing.T) {
	_, err := deriveData(t, oneZone)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 availability zones")
}
