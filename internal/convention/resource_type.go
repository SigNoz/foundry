package convention

import (
	"strconv"
)

// resourceType is what a derived name says the thing is, and the ordered
// qualifiers that narrow it. Adding a resource is one var entry below.
type resourceType struct {
	short      string
	qualifiers []qualifier
}

var (
	typeCluster         = resourceType{short: "cls"}
	typeVPC             = resourceType{short: "vpc"}
	typeInternetGateway = resourceType{short: "igw"}
	typeSubnet          = resourceType{short: "sub", qualifiers: []qualifier{qualifierVisibility, qualifierZone}}
	typeRouteTable      = resourceType{short: "rt", qualifiers: []qualifier{qualifierVisibility, qualifierZone}}
	typeNATGateway      = resourceType{short: "nat", qualifiers: []qualifier{qualifierZone}}
	typeSecurityGroup   = resourceType{short: "sg", qualifiers: []qualifier{qualifierRole}}
	typeRole            = resourceType{short: "iam", qualifiers: []qualifier{qualifierRole}}
	typeNode            = resourceType{short: "node", qualifiers: []qualifier{qualifierStorage, qualifierOrdinal}}
	typeVolume          = resourceType{short: "vol", qualifiers: []qualifier{qualifierStorage, qualifierOrdinal}}
)

func (resource resourceType) String() string {
	return resource.short
}

// qualifier renders one axis into a name segment. An empty string drops the
// segment, so one route table declaration serves both the zonal and shared forms.
type qualifier struct {
	of func(Resource) string
}

var (
	qualifierVisibility = qualifier{of: func(r Resource) string { return r.visibility.Short() }}
	qualifierZone       = qualifier{of: func(r Resource) string { return r.zone.Short() }}
	qualifierRole       = qualifier{of: func(r Resource) string { return r.role.String() }}
	qualifierStorage    = qualifier{of: func(r Resource) string { return r.storage.String() }}

	// Only types that have an ordinal declare it, so zero renders as "0".
	qualifierOrdinal = qualifier{of: func(r Resource) string { return strconv.Itoa(r.ordinal) }}
)
