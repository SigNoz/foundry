package convention

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/stretchr/testify/assert"
)

// Two types sharing a short form would derive the same name shape.
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

// An empty qualifier drops its segment, so one route table declaration serves
// both the zonal and the zone-shared form.
func TestUnsetQualifierDropsFromTheName(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	zone := MustParseZone("us-east-1a")

	assert.Equal(t, "foundry-rt-pub", substrate.RouteTable(VisibilityPublic).Name())
	assert.Equal(t, "foundry-rt-prv-east1a", substrate.RouteTableInZone(VisibilityPrivate, zone).Name())
}

// A declared qualifier that renders nothing would silently drop a segment meant
// to distinguish the name.
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

// Ordinal zero is a real ordinal, so only types that have one declare it.
func TestOrdinalZeroRenders(t *testing.T) {
	substrate := MustNewSubstrate("foundry")
	persistent := infrastructure.StorageClassPersistent

	assert.Equal(t, "foundry-node-persistent-0", substrate.Node(persistent, 0).Name())
	assert.Equal(t, "foundry-vpc", substrate.VPC().Name())
}
