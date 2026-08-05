package convention

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/stretchr/testify/assert"
)

// Two resource types sharing a short form would derive the same name shape, and
// a consumer reading a console could not tell them apart.
func TestResourceTypeShortFormsAreDistinct(t *testing.T) {
	resourceTypes := []resourceType{
		typeCluster, typeVPC, typeInternetGateway, typeSubnet, typeRouteTable,
		typeNATGateway, typeSecurityGroup, typeRole, typeNode, typeVolume,
	}

	seen := make(map[string]struct{}, len(resourceTypes))
	for _, resource := range resourceTypes {
		assert.NotContains(t, seen, resource.String())
		seen[resource.String()] = struct{}{}
	}
}

// A qualifier that renders empty drops out, which is what lets one route table
// declaration serve both the zonal private tables and the zone-shared public
// one. Without it they would be two variants for one AWS resource type.
func TestUnsetQualifierDropsFromTheName(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	zone := MustParseZone("us-east-1a")

	assert.Equal(t, "foundry-rt-pub", substrate.RouteTable(VisibilityPublic).Name())
	assert.Equal(t, "foundry-rt-prv-east1a", substrate.RouteTableInZone(VisibilityPrivate, zone).Name())
}

// Every declared qualifier has to contribute for the constructor that uses the
// type, or a name silently loses a segment that was meant to distinguish it.
func TestEveryDeclaredQualifierContributes(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	zone := MustParseZone("us-east-1a")
	persistent := infrastructure.StorageClassPersistent

	tests := []struct {
		name             string
		resource         Resource
		expectedSegments int
	}{
		{name: "VPC_NoQualifier", resource: substrate.VPC(), expectedSegments: 0},
		{name: "Subnet_VisibilityAndZone", resource: substrate.Subnet(VisibilityPrivate, zone), expectedSegments: 2},
		{name: "NATGateway_Zone", resource: substrate.NATGateway(zone), expectedSegments: 1},
		{name: "IAMRole_Role", resource: substrate.Role(RoleExec), expectedSegments: 1},
		{name: "Node_ClassAndOrdinal", resource: substrate.Node(persistent, 0), expectedSegments: 2},
		{name: "Volume_ClassAndOrdinal", resource: substrate.Volume(persistent, 0), expectedSegments: 2},
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

// Ordinal zero is a real ordinal, not an absent one, so only the types that have
// one declare the qualifier -- otherwise every other resource would render a
// stray "0".
func TestOrdinalZeroRenders(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	persistent := infrastructure.StorageClassPersistent

	assert.Equal(t, "foundry-node-persistent-0", substrate.Node(persistent, 0).Name())
	assert.Equal(t, "foundry-vpc", substrate.VPC().Name())
}
