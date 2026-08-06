package aws

import (
	"github.com/signoz/foundry/internal/convention"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
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

// derive renders a declaration written the way the molding settles it.
func derive(t *testing.T, declaration string) (*infrastructure.ResourceConfigResources, error) {
	t.Helper()

	config := &infrastructure.ResourceConfig{}
	require.NoError(t, domain.UnmarshalYAML([]byte(declaration), config))

	return Resources(convention.MustNewSubstrate("foundry"), config, nil)
}

func mustDerive(t *testing.T, declaration string) *infrastructure.ResourceConfigResources {
	t.Helper()

	derived, err := derive(t, declaration)
	require.NoError(t, err)

	return derived
}

// These names reach live infrastructure. Changing one replaces the resource it
// belongs to rather than updating it.
func TestResources(t *testing.T) {
	derived := mustDerive(t, oneZone)

	tests := []struct {
		name         string
		of           func(*infrastructure.ResourceConfigResources) string
		expectedName string
	}{
		{name: "Cluster_Unqualified", of: func(r *infrastructure.ResourceConfigResources) string { return r.Cluster.Name }, expectedName: "foundry-cls"},
		{name: "VPC_Unqualified", of: func(r *infrastructure.ResourceConfigResources) string { return r.VPC.Name }, expectedName: "foundry-vpc"},
		{name: "InternetGateway_Unqualified", of: func(r *infrastructure.ResourceConfigResources) string { return r.InternetGateway.Name }, expectedName: "foundry-igw"},
		{name: "Subnet_Keyed", of: func(r *infrastructure.ResourceConfigResources) string { return r.Subnets["private-a"].Name }, expectedName: "foundry-sub-private-a"},
		{name: "RouteTable_Keyed", of: func(r *infrastructure.ResourceConfigResources) string { return r.RouteTables["public-a"].Name }, expectedName: "foundry-rt-public-a"},
		{name: "NATGateway_KeyedByTheSubnetItServes", of: func(r *infrastructure.ResourceConfigResources) string { return r.NATGateways["private-a"].Name }, expectedName: "foundry-nat-private-a"},
		{name: "ElasticIP_KeyedByTheSubnetItServes", of: func(r *infrastructure.ResourceConfigResources) string { return r.NATGateways["private-a"].Address.Name }, expectedName: "foundry-eip-private-a"},
		{name: "SecurityGroup_Role", of: func(r *infrastructure.ResourceConfigResources) string { return r.SecurityGroup.Name }, expectedName: "foundry-sg-task"},
		{name: "SecurityGroupRule_RoleAndPurpose", of: func(r *infrastructure.ResourceConfigResources) string {
			return r.SecurityGroupRules["intra-cluster"].Name
		}, expectedName: "foundry-sg-task-intra-cluster"},
		{name: "Role_NodeOnly", of: func(r *infrastructure.ResourceConfigResources) string { return r.Roles["node"].Name }, expectedName: "foundry-iam-node"},
		{name: "InstanceProfile_Role", of: func(r *infrastructure.ResourceConfigResources) string { return r.InstanceProfile.Name }, expectedName: "foundry-prf-node"},
		{name: "LaunchTemplate_GroupKey", of: func(r *infrastructure.ResourceConfigResources) string {
			return r.InstanceGroups["ephemeral"].LaunchTemplate.Name
		}, expectedName: "foundry-lt-ephemeral"},
		{name: "AutoscalingGroup_GroupKey", of: func(r *infrastructure.ResourceConfigResources) string {
			return r.InstanceGroups["ephemeral"].AutoscalingGroup.Name
		}, expectedName: "foundry-asg-ephemeral"},
		{name: "Node_GroupKeyAndOrdinal", of: func(r *infrastructure.ResourceConfigResources) string {
			return r.InstanceGroups["persistent"].Nodes[0].Name
		}, expectedName: "foundry-node-persistent-0"},
		{name: "Volume_GroupKeyAndOrdinal", of: func(r *infrastructure.ResourceConfigResources) string {
			return r.InstanceGroups["persistent"].Nodes[0].Volume.Name
		}, expectedName: "foundry-vol-persistent-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, tt.of(derived))
		})
	}
}

// The tags a consuming casting filters on are the only ones whose spelling live
// infrastructure depends on.
func TestResourcesContractTags(t *testing.T) {
	derived := mustDerive(t, oneZone)

	assert.Equal(t, "private", derived.Subnets["private-a"].Tags[Tag(convention.TagKeySubnetType)])
	assert.Equal(t, "public", derived.Subnets["public-a"].Tags[Tag(convention.TagKeySubnetType)])
	assert.Equal(t, "persistent", derived.InstanceGroups["persistent"].Nodes[0].Tags[Tag(convention.TagKeyStorage)])
	assert.Equal(t, "persistent", derived.InstanceGroups["persistent"].Nodes[0].Volume.Tags[Tag(convention.TagKeyStorage)])

	assert.Equal(t, map[string]string{
		Tag(convention.TagKeyName):    "foundry",
		Tag(convention.TagKeyStorage): "persistent",
	}, derived.InstanceGroups["persistent"].Selector)

	// The claim tag is stamped after provisioning, so a casting that reconciles
	// has to be told to leave it alone.
	assert.Equal(t, []string{Tag(convention.TagKeyIdentities)}, derived.IgnoredTags)
}

// A group's selector has to match what its own nodes are stamped with, or the
// substrate advertises a placement nothing satisfies.
func TestResourcesSelectorMatchesItsNodes(t *testing.T) {
	group := mustDerive(t, oneZone).InstanceGroups["persistent"]

	for _, node := range group.Nodes {
		for key, value := range group.Selector {
			assert.Equal(t, value, node.Tags[key], "node %s does not match the group selector on %s", node.Name, key)
		}
	}
}

// Foundry's ownership labels and the derived tags are what a consumer matches
// on, so an operator's own tags sit underneath rather than over them.
func TestResourcesCloudLabelsDoNotOverrideTheContract(t *testing.T) {
	derived := mustDerive(t, oneZone+"cloudLabels:\n  team: observability\n  "+Tag(convention.TagKeyName)+": not-foundry\n")

	assert.Equal(t, "observability", derived.VPC.Tags["team"])
	assert.Equal(t, "foundry", derived.VPC.Tags[Tag(convention.TagKeyName)])
}

// Nodes are laid out across the group's subnets in ordinal order, and a node's
// volume goes wherever the node does.
func TestResourcesPlacementCyclesThroughSubnets(t *testing.T) {
	derived := mustDerive(t, twoZones)

	group := derived.InstanceGroups["persistent"]
	assert.Equal(t, []string{"private-a", "private-b"}, group.Subnets)
	assert.Equal(t, []string{"private-a", "private-b", "private-a"}, []string{
		group.Nodes[0].Subnet, group.Nodes[1].Subnet, group.Nodes[2].Subnet,
	})

	// Each zone's gateway sits in a public subnet of that same zone.
	assert.Equal(t, "public-a", derived.NATGateways["private-a"].Subnet)
	assert.Equal(t, "public-b", derived.NATGateways["private-b"].Subnet)
}

// A group that names its own subnets is placed only there.
func TestResourcesHonourStatedPlacement(t *testing.T) {
	group := mustDerive(t, twoZones+"  persistent:\n    subnets: [private-b]\n").InstanceGroups["persistent"]

	for _, node := range group.Nodes {
		assert.Equal(t, "private-b", node.Subnet)
	}
}

// An adopted network is referenced, never described: foundry adds no gateway,
// no route table and no tags to something it does not own.
func TestResourcesAdoptedNetwork(t *testing.T) {
	derived := mustDerive(t, `networking:
  networkID: vpc-0a1b2c3d
  subnets:
    private-a: {type: private, zone: us-east-1a, id: subnet-0a1b2c3d}
instanceGroups:
  persistent: {storage: persistent, machineType: m5.large, minSize: 3, maxSize: 3}
`)

	assert.Equal(t, "vpc-0a1b2c3d", derived.VPC.ID)
	assert.Empty(t, derived.VPC.Name)
	assert.Empty(t, derived.VPC.Tags)
	assert.Empty(t, derived.InternetGateway.Name)
	assert.Empty(t, derived.RouteTables)
	assert.Empty(t, derived.NATGateways)

	assert.Equal(t, "subnet-0a1b2c3d", derived.Subnets["private-a"].ID)
	assert.Empty(t, derived.Subnets["private-a"].Name)

	// The compute placed in it is still foundry's.
	assert.Equal(t, "foundry-node-persistent-0", derived.InstanceGroups["persistent"].Nodes[0].Name)
}

// A private subnet that already routes somewhere gets no gateway of its own,
// and the id it routes through is carried through for the casting to reference.
func TestResourcesAdoptedEgress(t *testing.T) {
	derived := mustDerive(t, `networking:
  networkCIDR: 10.0.0.0/16
  subnets:
    private-b: {type: private, zone: us-east-1b, cidr: 10.0.32.0/19, egress: nat-0a1b2c3d}
instanceGroups:
  ephemeral: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1}
`)

	gateway := derived.NATGateways["private-b"]
	assert.Equal(t, "nat-0a1b2c3d", gateway.ID)
	assert.Empty(t, gateway.Name)
	assert.Nil(t, gateway.Address)

	// No public subnet was declared, so nothing routes to a gateway foundry owns.
	assert.Empty(t, derived.InternetGateway.Name)
}

// A group with nowhere to go would provision nodes no workload can be placed
// on, so it fails instead.
func TestResourcesWithoutAPrivateSubnet(t *testing.T) {
	_, err := derive(t, `networking:
  subnets:
    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}
instanceGroups:
  ephemeral: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1}
`)

	assert.Error(t, err)
}
