package ecs

import (
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
	derived := mustDeriveData(t, oneZone)

	tests := []struct {
		name         string
		of           func(*Resources) string
		expectedName string
	}{
		{name: "Cluster_Unqualified", of: func(d *Resources) string { return d.Cluster.Name }, expectedName: "foundry-cls"},
		{name: "VPC_Unqualified", of: func(d *Resources) string { return d.Network.VPC.Name }, expectedName: "foundry-vpc"},
		{name: "InternetGateway_Unqualified", of: func(d *Resources) string { return d.Network.InternetGateway.Name }, expectedName: "foundry-igw"},
		{name: "Subnet_Keyed", of: func(d *Resources) string { return d.Network.Subnets["private-a"].Name }, expectedName: "foundry-sub-private-a"},
		{name: "RouteTable_Keyed", of: func(d *Resources) string { return d.Network.RouteTables["public-a"].Name }, expectedName: "foundry-rt-public-a"},
		{name: "NATGateway_KeyedByTheSubnetItServes", of: func(d *Resources) string { return d.Network.NATGateways["private-a"].Name }, expectedName: "foundry-nat-private-a"},
		{name: "ElasticIP_KeyedByTheSubnetItServes", of: func(d *Resources) string { return d.Network.NATGateways["private-a"].Address.Name }, expectedName: "foundry-eip-private-a"},
		{name: "SecurityGroup_Role", of: func(d *Resources) string { return d.SecurityGroup.Name }, expectedName: "foundry-sg-task"},
		{name: "SecurityGroupRule_RoleAndPurpose", of: func(d *Resources) string {
			return d.SecurityGroupRules["intra-cluster"].Name
		}, expectedName: "foundry-sg-task-intra-cluster"},
		{name: "Role_NodeOnly", of: func(d *Resources) string { return d.Roles["node"].Name }, expectedName: "foundry-iam-node"},
		{name: "InstanceProfile_Role", of: func(d *Resources) string { return d.InstanceProfile.Name }, expectedName: "foundry-prf-node"},
		{name: "LaunchTemplate_GroupKey", of: func(d *Resources) string {
			return d.Pools["ephemeral"].LaunchTemplate.Name
		}, expectedName: "foundry-lt-ephemeral"},
		{name: "AutoscalingGroup_GroupKey", of: func(d *Resources) string {
			return d.Pools["ephemeral"].AutoscalingGroup.Name
		}, expectedName: "foundry-asg-ephemeral"},
		{name: "Node_GroupKeyAndOrdinal", of: func(d *Resources) string {
			return d.Pinned["persistent"].Nodes[0].Name
		}, expectedName: "foundry-node-persistent-0"},
		{name: "Volume_GroupKeyAndOrdinal", of: func(d *Resources) string {
			return d.Pinned["persistent"].Nodes[0].Volume.Name
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
func TestTemplateDataContractTags(t *testing.T) {
	derived := mustDeriveData(t, oneZone)

	assert.Equal(t, "private", derived.Network.Subnets["private-a"].Tags[aws.Tag(contract.TagKeySubnetType)])
	assert.Equal(t, "public", derived.Network.Subnets["public-a"].Tags[aws.Tag(contract.TagKeySubnetType)])
	assert.Equal(t, "persistent", derived.Pinned["persistent"].Nodes[0].Tags[aws.Tag(contract.TagKeyStorage)])
	assert.Equal(t, "persistent", derived.Pinned["persistent"].Nodes[0].Volume.Tags[aws.Tag(contract.TagKeyStorage)])

	assert.Equal(t, map[string]string{
		aws.Tag(contract.TagKeyName):    "foundry",
		aws.Tag(contract.TagKeyStorage): "persistent",
	}, derived.Pinned["persistent"].Selector)

	// The claim tag is stamped after provisioning, so a casting that reconciles
	// has to be told to leave it alone.
	assert.Equal(t, []string{aws.Tag(contract.TagKeyIdentities)}, derived.IgnoredTags)
}

// A group's selector has to match what its own nodes are stamped with, or the
// substrate advertises a placement nothing satisfies.
func TestTemplateDataSelectorMatchesItsNodes(t *testing.T) {
	group := mustDeriveData(t, oneZone).Pinned["persistent"]

	for _, node := range group.Nodes {
		for key, value := range group.Selector {
			assert.Equal(t, value, node.Tags[key], "node %s does not match the group selector on %s", node.Name, key)
		}
	}
}

// Foundry's ownership labels and the derived tags are what a consumer matches
// on, so an operator's own tags sit underneath rather than over them.
func TestTemplateDataCloudLabelsDoNotOverrideTheContract(t *testing.T) {
	derived := mustDeriveData(t, oneZone+"cloudLabels:\n  team: observability\n  "+aws.Tag(contract.TagKeyName)+": not-foundry\n")

	assert.Equal(t, "observability", derived.Network.VPC.Tags["team"])
	assert.Equal(t, "foundry", derived.Network.VPC.Tags[aws.Tag(contract.TagKeyName)])
}

// Nodes are laid out across the group's subnets in ordinal order, and a node's
// volume goes wherever the node does.
func TestTemplateDataPlacementCyclesThroughSubnets(t *testing.T) {
	derived := mustDeriveData(t, twoZones)

	group := derived.Pinned["persistent"]
	assert.Equal(t, []string{"private-a", "private-b", "private-a"}, []string{
		group.Nodes[0].Subnet, group.Nodes[1].Subnet, group.Nodes[2].Subnet,
	})

	// Each zone's gateway sits in a public subnet of that same zone.
	assert.Equal(t, "public-a", derived.Network.NATGateways["private-a"].Subnet)
	assert.Equal(t, "public-b", derived.Network.NATGateways["private-b"].Subnet)
}

// A group that names its own subnets is placed only there.
func TestTemplateDataHonourStatedPlacement(t *testing.T) {
	group := mustDeriveData(t, twoZones+"  persistent: {storage: persistent, machineType: m5.large, minSize: 3, maxSize: 3, subnets: [private-b]}\n").Pinned["persistent"]

	assert.Len(t, group.Nodes, 3)

	for _, node := range group.Nodes {
		assert.Equal(t, "private-b", node.Subnet)
	}
}

// An adopted network is referenced, never described: foundry adds no gateway,
// no route table and no tags to something it does not own.
func TestTemplateDataAdoptedNetwork(t *testing.T) {
	derived := mustDeriveData(t, `networking:
  networkID: vpc-0a1b2c3d
  subnets:
    private-a: {type: private, zone: us-east-1a, id: subnet-0a1b2c3d}
instanceGroups:
  persistent: {storage: persistent, machineType: m5.large, minSize: 3, maxSize: 3}
`)

	assert.Equal(t, "vpc-0a1b2c3d", derived.Network.VPC.ID)
	assert.Empty(t, derived.Network.VPC.Name)
	assert.Empty(t, derived.Network.VPC.Tags)
	assert.Empty(t, derived.Network.InternetGateway.Name)
	assert.Empty(t, derived.Network.RouteTables)
	assert.Empty(t, derived.Network.NATGateways)

	assert.Equal(t, "subnet-0a1b2c3d", derived.Network.Subnets["private-a"].ID)
	assert.Empty(t, derived.Network.Subnets["private-a"].Name)

	// The compute placed in it is still foundry's.
	assert.Equal(t, "foundry-node-persistent-0", derived.Pinned["persistent"].Nodes[0].Name)
}

// A private subnet that already routes somewhere gets no gateway of its own,
// and the id it routes through is carried through for the casting to reference.
func TestTemplateDataAdoptedEgress(t *testing.T) {
	derived := mustDeriveData(t, `networking:
  networkCIDR: 10.0.0.0/16
  subnets:
    private-b: {type: private, zone: us-east-1b, cidr: 10.0.32.0/19, egress: nat-0a1b2c3d}
instanceGroups:
  ephemeral: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1}
`)

	gateway := derived.Network.NATGateways["private-b"]
	assert.Equal(t, "nat-0a1b2c3d", gateway.ID)
	assert.Empty(t, gateway.Name)
	assert.Nil(t, gateway.Address)

	// No public subnet was declared, so nothing routes to a gateway foundry owns.
	assert.Empty(t, derived.Network.InternetGateway.Name)
}

// A group with nowhere to go would provision nodes no workload can be placed
// on, so it fails instead.
func TestTemplateDataWithoutAPrivateSubnet(t *testing.T) {
	_, err := deriveData(t, `networking:
  subnets:
    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}
instanceGroups:
  ephemeral: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1}
`)

	assert.Error(t, err)
}
