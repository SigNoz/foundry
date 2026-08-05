package convention

import (
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
)

// Resource is one thing a substrate provisions, described once. Its name, its
// tags, and the selection that finds it are all derived from that description, so
// a fact stated once cannot render two ways.
//
// Callers use the constructor for what they are provisioning; it supplies the
// resource type.
type Resource struct {
	substrate    Substrate
	resourceType resourceType

	visibility Visibility
	storage    infrastructure.StorageClass
	zone       Zone
	role       Role
	ordinal    int

	ownership  Ownership
	kind       infrastructure.ResourceKind
	identities Identities
}

func (s Substrate) Cluster() Resource {
	return Resource{substrate: s, resourceType: typeCluster}
}

func (s Substrate) VPC() Resource {
	return Resource{substrate: s, resourceType: typeVPC}
}

func (s Substrate) InternetGateway() Resource {
	return Resource{substrate: s, resourceType: typeInternetGateway}
}

func (s Substrate) Subnet(visibility Visibility, zone Zone) Resource {
	return Resource{substrate: s, resourceType: typeSubnet, visibility: visibility, zone: zone}
}

// RouteTable is a table shared across zones, which one internet gateway serves.
func (s Substrate) RouteTable(visibility Visibility) Resource {
	return Resource{substrate: s, resourceType: typeRouteTable, visibility: visibility}
}

// RouteTableInZone is a table per zone, each routing to that zone's NAT gateway.
func (s Substrate) RouteTableInZone(visibility Visibility, zone Zone) Resource {
	return Resource{substrate: s, resourceType: typeRouteTable, visibility: visibility, zone: zone}
}

func (s Substrate) NATGateway(zone Zone) Resource {
	return Resource{substrate: s, resourceType: typeNATGateway, zone: zone}
}

func (s Substrate) SecurityGroup(role Role) Resource {
	return Resource{substrate: s, resourceType: typeSecurityGroup, role: role}
}

func (s Substrate) Role(role Role) Resource {
	return Resource{substrate: s, resourceType: typeRole, role: role}
}

// Node and Volume take the storage class, which is a node group's whole
// identity: a consuming casting selects by class and cannot name a group.
func (s Substrate) Node(storage infrastructure.StorageClass, ordinal int) Resource {
	return Resource{substrate: s, resourceType: typeNode, storage: storage, ordinal: ordinal}
}

func (s Substrate) Volume(storage infrastructure.StorageClass, ordinal int) Resource {
	return Resource{substrate: s, resourceType: typeVolume, storage: storage, ordinal: ordinal}
}

// WithOwnership marks a resource adopted rather than created. It changes no name.
func (r Resource) WithOwnership(ownership Ownership) Resource {
	r.ownership = ownership

	return r
}

// WithKind records the Kind the substrate is provisioned for.
func (r Resource) WithKind(kind infrastructure.ResourceKind) Resource {
	r.kind = kind

	return r
}

// WithClaims records the identities holding a volume. See Identities.
func (r Resource) WithClaims(identities Identities) Resource {
	r.identities = identities

	return r
}

// Name is <substrate>-<type>[-<qualifier>...], broad to narrow so a substrate's
// resources share a prefix and sort together. It fills a provider's name argument
// where one exists, and the display tag always -- an instance or a volume has no
// name of its own. Which qualifiers apply is the resource type's declaration.
func (r Resource) Name() string {
	parts := make([]string, 0, len(r.resourceType.qualifiers)+2)
	parts = append(parts, r.substrate.name, r.resourceType.String())

	for _, qualifier := range r.resourceType.qualifiers {
		if segment := qualifier.of(r); segment != "" {
			parts = append(parts, segment)
		}
	}

	return strings.Join(parts, "-")
}

// Selection is the set that finds exactly this resource.
func (r Resource) Selection() Selection {
	return Selection{substrate: r.substrate, storage: r.storage, identities: r.identities}
}

// stamp is the selection's tags plus the provenance nothing reads back. Each
// check is on whether an axis applies, not on which resource this is.
func (r Resource) stamp() Tags {
	tags := r.Selection().match()

	tags = append(tags, Tag{Key: TagKeyOwner, Value: r.ownership.String()})

	// An adopted resource keeps the name it already had.
	if !r.ownership.IsShared() {
		tags = append(tags, Tag{Key: TagKeyDisplayName, Value: r.Name()})
	}

	if r.kind != (infrastructure.ResourceKind{}) {
		tags = append(tags, Tag{Key: TagKeyResourceKind, Value: r.kind.String()})
	}

	if r.visibility != (Visibility{}) {
		tags = append(tags, Tag{Key: TagKeyVisibility, Value: r.visibility.String()})
	}

	return tags
}

// Tags is every tag this resource carries. Ownership labels are a separate
// family: a casting merges CastingMeta.Labels() in alongside these.
func (r Resource) Tags() map[string]string {
	return r.stamp().Map()
}

// Filter is the tag match that finds this resource.
func (r Resource) Filter() map[string]string {
	return r.Selection().Filter()
}
