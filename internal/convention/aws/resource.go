package aws

import (
	"github.com/signoz/foundry/internal/convention"
	"strconv"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
)

// Resource is one thing a substrate provisions. Its name, tags and selection
// all derive from one description. Use the constructor for what you are
// provisioning.
type Resource struct {
	substrate    convention.Substrate
	resourceType resourceType

	key        convention.Key
	purpose    convention.Key
	subnetType v1alpha1.SubnetType
	storage    v1alpha1.StorageClass
	role       Role
	ordinal    int

	ownership  convention.Ownership
	identities convention.Identities
}

func Cluster(s convention.Substrate) Resource {
	return Resource{substrate: s, resourceType: typeCluster}
}

func VPC(s convention.Substrate) Resource {
	return Resource{substrate: s, resourceType: typeVPC}
}

func InternetGateway(s convention.Substrate) Resource {
	return Resource{substrate: s, resourceType: typeInternetGateway}
}

// Subnet takes its type separately from its key. The key is the operator's own
// reference and says nothing a consumer can rely on.
func Subnet(s convention.Substrate, key convention.Key, subnetType v1alpha1.SubnetType) Resource {
	return Resource{substrate: s, resourceType: typeSubnet, key: key, subnetType: subnetType}
}

func RouteTable(s convention.Substrate, key convention.Key) Resource {
	return Resource{substrate: s, resourceType: typeRouteTable, key: key}
}

func NATGateway(s convention.Substrate, key convention.Key) Resource {
	return Resource{substrate: s, resourceType: typeNATGateway, key: key}
}

func ElasticIP(s convention.Substrate, key convention.Key) Resource {
	return Resource{substrate: s, resourceType: typeElasticIP, key: key}
}

func SecurityGroup(s convention.Substrate, role Role) Resource {
	return Resource{substrate: s, resourceType: typeSecurityGroup, role: role}
}

// SecurityGroupRule is one rule of the group for the same role, distinguished
// by what it admits.
func SecurityGroupRule(s convention.Substrate, role Role, purpose convention.Key) Resource {
	return Resource{substrate: s, resourceType: typeSecurityGroup, role: role, purpose: purpose}
}

func IAMRole(s convention.Substrate, role Role) Resource {
	return Resource{substrate: s, resourceType: typeRole, role: role}
}

// RolePolicy is a policy inline on a role, distinguished by what it grants.
func IAMRolePolicy(s convention.Substrate, role Role, purpose convention.Key) Resource {
	return Resource{substrate: s, resourceType: typeRole, role: role, purpose: purpose}
}

func InstanceProfile(s convention.Substrate, role Role) Resource {
	return Resource{substrate: s, resourceType: typeInstanceProfile, role: role}
}

func LaunchTemplate(s convention.Substrate, group convention.NodeGroup) Resource {
	return Resource{substrate: s, resourceType: typeLaunchTemplate, key: group.Key(), storage: group.Storage()}
}

func AutoscalingGroup(s convention.Substrate, group convention.NodeGroup) Resource {
	return Resource{substrate: s, resourceType: typeAutoscalingGroup, key: group.Key(), storage: group.Storage()}
}

func Node(s convention.Substrate, group convention.NodeGroup, ordinal int) Resource {
	return Resource{substrate: s, resourceType: typeNode, key: group.Key(), storage: group.Storage(), ordinal: ordinal}
}

func Volume(s convention.Substrate, group convention.NodeGroup, ordinal int) Resource {
	return Resource{substrate: s, resourceType: typeVolume, key: group.Key(), storage: group.Storage(), ordinal: ordinal}
}

// WithOwnership marks a resource adopted rather than created. It changes no name.
func (r Resource) WithOwnership(ownership convention.Ownership) Resource {
	r.ownership = ownership

	return r
}

// WithClaims records the identities holding a volume. See convention.Identities.
func (r Resource) WithClaims(identities convention.Identities) Resource {
	r.identities = identities

	return r
}

// Name is <substrate>-<type>[-<qualifier>...], broad to narrow. It fills a
// provider's name argument where one exists and the display tag always. An
// instance has no name of its own.
func (r Resource) Name() string {
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
func (r Resource) Selection() convention.Selection {
	return r.substrate.Select().WithSubnetType(r.subnetType).WithStorage(r.storage).WithClaims(r.identities)
}

// stamp is the selection's tags plus the provenance nothing reads back.
func (r Resource) stamp() map[string]string {
	tags := Filter(r.Selection())
	tags[Tag(convention.TagKeyOwner)] = r.ownership.String()

	// An adopted resource keeps the name it already had.
	if !r.ownership.IsShared() {
		tags[displayName] = r.Name()
	}

	return tags
}

// Tags is every tag this resource carries. A casting merges
// CastingMeta.Labels() in alongside these.
func (r Resource) Tags() map[string]string {
	return r.stamp()
}

// Filter is the tag match that finds this resource.
func (r Resource) Filter() map[string]string {
	return Filter(r.Selection())
}

// resourceType is what a derived name says the thing is, and the ordered
// qualifiers that narrow it. Adding a resource is one var entry below.
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
	typeNode             = resourceType{short: "node", qualifiers: []qualifier{qualifierKey, qualifierOrdinal}}
	typeVolume           = resourceType{short: "vol", qualifiers: []qualifier{qualifierKey, qualifierOrdinal}}
)

func (resource resourceType) String() string {
	return resource.short
}

// qualifier renders one axis into a name segment. An empty string drops the
// segment: one declaration serves a security group and its rules.
type qualifier struct {
	of func(Resource) string
}

var (
	qualifierKey     = qualifier{of: func(r Resource) string { return r.key.String() }}
	qualifierRole    = qualifier{of: func(r Resource) string { return r.role.String() }}
	qualifierPurpose = qualifier{of: func(r Resource) string { return r.purpose.String() }}

	// Only types that have an ordinal declare it. Zero renders as "0".
	qualifierOrdinal = qualifier{of: func(r Resource) string { return strconv.Itoa(r.ordinal) }}
)
