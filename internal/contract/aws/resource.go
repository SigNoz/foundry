package aws

import (
	"github.com/signoz/foundry/internal/contract"
	"strconv"
	"strings"
)

// Descriptor describes one thing a substrate provisions; its name, tags and
// selection all derive from it. Each resource type has its own constructor.
type Descriptor struct {
	substrate    contract.Substrate
	resourceType resourceType

	key        contract.Key
	purpose    contract.Key
	subnetType contract.SubnetType
	storage    contract.StorageClass
	role       Role
	ordinal    int

	ownership  contract.Ownership
	identities contract.Identities
}

func Cluster(s contract.Substrate) Descriptor {
	return Descriptor{substrate: s, resourceType: typeCluster}
}

func VPC(s contract.Substrate) Descriptor {
	return Descriptor{substrate: s, resourceType: typeVPC}
}

func InternetGateway(s contract.Substrate) Descriptor {
	return Descriptor{substrate: s, resourceType: typeInternetGateway}
}

// Subnet takes its type separately from its key. The key is the operator's own
// reference and says nothing a consumer can rely on.
func Subnet(s contract.Substrate, key contract.Key, subnetType contract.SubnetType) Descriptor {
	return Descriptor{substrate: s, resourceType: typeSubnet, key: key, subnetType: subnetType}
}

func RouteTable(s contract.Substrate, key contract.Key) Descriptor {
	return Descriptor{substrate: s, resourceType: typeRouteTable, key: key}
}

func NATGateway(s contract.Substrate, key contract.Key) Descriptor {
	return Descriptor{substrate: s, resourceType: typeNATGateway, key: key}
}

func ElasticIP(s contract.Substrate, key contract.Key) Descriptor {
	return Descriptor{substrate: s, resourceType: typeElasticIP, key: key}
}

func SecurityGroup(s contract.Substrate, role Role) Descriptor {
	return Descriptor{substrate: s, resourceType: typeSecurityGroup, role: role}
}

// SecurityGroupRule is one rule of the group for the same role, distinguished
// by what it admits.
func SecurityGroupRule(s contract.Substrate, role Role, purpose contract.Key) Descriptor {
	return Descriptor{substrate: s, resourceType: typeSecurityGroup, role: role, purpose: purpose}
}

func IAMRole(s contract.Substrate, role Role) Descriptor {
	return Descriptor{substrate: s, resourceType: typeRole, role: role}
}

// IAMRolePolicy is a policy inline on a role, distinguished by what it grants.
func IAMRolePolicy(s contract.Substrate, role Role, purpose contract.Key) Descriptor {
	return Descriptor{substrate: s, resourceType: typeRole, role: role, purpose: purpose}
}

func InstanceProfile(s contract.Substrate, role Role) Descriptor {
	return Descriptor{substrate: s, resourceType: typeInstanceProfile, role: role}
}

func LaunchTemplate(s contract.Substrate, group contract.NodeGroup) Descriptor {
	return Descriptor{substrate: s, resourceType: typeLaunchTemplate, key: group.Key(), storage: group.Storage()}
}

func AutoscalingGroup(s contract.Substrate, group contract.NodeGroup) Descriptor {
	return Descriptor{substrate: s, resourceType: typeAutoscalingGroup, key: group.Key(), storage: group.Storage()}
}

// NodeGroup is a pool the provider scales and replaces nodes in on the
// substrate's behalf, holding no name for any node of its own.
func NodeGroup(s contract.Substrate, group contract.NodeGroup) Descriptor {
	return Descriptor{substrate: s, resourceType: typeNodeGroup, key: group.Key(), storage: group.Storage()}
}

func Node(s contract.Substrate, group contract.NodeGroup, ordinal int) Descriptor {
	return Descriptor{substrate: s, resourceType: typeNode, key: group.Key(), storage: group.Storage(), ordinal: ordinal}
}

func Volume(s contract.Substrate, group contract.NodeGroup, ordinal int) Descriptor {
	return Descriptor{substrate: s, resourceType: typeVolume, key: group.Key(), storage: group.Storage(), ordinal: ordinal}
}

// WithOwnership marks a resource adopted rather than created. The derived name
// is unaffected.
func (r Descriptor) WithOwnership(ownership contract.Ownership) Descriptor {
	r.ownership = ownership

	return r
}

// WithClaims records the identities holding a volume.
func (r Descriptor) WithClaims(identities contract.Identities) Descriptor {
	r.identities = identities

	return r
}

// Name is <substrate>-<type>[-<qualifier>...]. A qualifier that does not apply
// to the resource type is left out.
func (r Descriptor) Name() string {
	parts := make([]string, 0, len(r.resourceType.qualifiers)+2)
	parts = append(parts, r.substrate.String(), r.resourceType.String())

	for _, qualifier := range r.resourceType.qualifiers {
		if segment := qualifier.of(r); segment != "" {
			parts = append(parts, segment)
		}
	}

	return strings.Join(parts, "-")
}

// Selection is the set that finds exactly this resource.
func (r Descriptor) Selection() contract.Selection {
	return r.substrate.Select().WithSubnetType(r.subnetType).WithStorage(r.storage).WithClaims(r.identities)
}

// stamp is the selection's tags plus ownership and the display name.
func (r Descriptor) stamp() map[string]string {
	tags := Filter(r.Selection())
	tags[Tag(contract.TagKeyOwner)] = r.ownership.String()

	// An adopted resource keeps the name it already had.
	if !r.ownership.IsShared() {
		tags[displayName] = r.Name()
	}

	return tags
}

// Tags is every tag this resource carries, before a casting merges
// CastingMeta.Labels() in alongside them.
func (r Descriptor) Tags() map[string]string {
	return r.stamp()
}

// Filter is the tag match that finds this resource.
func (r Descriptor) Filter() map[string]string {
	return Filter(r.Selection())
}

// resourceType is the type token in a derived name and the ordered qualifiers
// that follow it.
type resourceType struct {
	short      string
	qualifiers []qualifier
}

var (
	typeCluster          = resourceType{short: "cls"}
	typeVPC              = resourceType{short: "vpc"}
	typeInternetGateway  = resourceType{short: "igw"}
	typeSubnet           = resourceType{short: "sub", qualifiers: []qualifier{qualifierKey}}
	typeRouteTable       = resourceType{short: "rt", qualifiers: []qualifier{qualifierKey}}
	typeNATGateway       = resourceType{short: "nat", qualifiers: []qualifier{qualifierKey}}
	typeElasticIP        = resourceType{short: "eip", qualifiers: []qualifier{qualifierKey}}
	typeSecurityGroup    = resourceType{short: "sg", qualifiers: []qualifier{qualifierRole, qualifierPurpose}}
	typeRole             = resourceType{short: "iam", qualifiers: []qualifier{qualifierRole, qualifierPurpose}}
	typeInstanceProfile  = resourceType{short: "prf", qualifiers: []qualifier{qualifierRole}}
	typeLaunchTemplate   = resourceType{short: "lt", qualifiers: []qualifier{qualifierKey}}
	typeAutoscalingGroup = resourceType{short: "asg", qualifiers: []qualifier{qualifierKey}}
	typeNodeGroup        = resourceType{short: "ng", qualifiers: []qualifier{qualifierKey}}
	typeNode             = resourceType{short: "node", qualifiers: []qualifier{qualifierKey, qualifierOrdinal}}
	typeVolume           = resourceType{short: "vol", qualifiers: []qualifier{qualifierKey, qualifierOrdinal}}
)

func (resource resourceType) String() string {
	return resource.short
}

// qualifier renders one axis into a name segment. An empty string drops the
// segment, so a security group and its rules share one resource type.
type qualifier struct {
	of func(Descriptor) string
}

var (
	qualifierKey     = qualifier{of: func(r Descriptor) string { return r.key.String() }}
	qualifierRole    = qualifier{of: func(r Descriptor) string { return r.role.String() }}
	qualifierPurpose = qualifier{of: func(r Descriptor) string { return r.purpose.String() }}

	// Declared only by types that have an ordinal, since zero renders as "0".
	qualifierOrdinal = qualifier{of: func(r Descriptor) string { return strconv.Itoa(r.ordinal) }}
)
