package contract

import "github.com/signoz/foundry/internal/errors"

var (
	// SubnetTypePrivate subnets have no route to an internet gateway. Workloads
	// go here; egress, where a subnet needs it, is a NAT gateway's job.
	SubnetTypePrivate = SubnetType{s: "private"}

	// SubnetTypePublic subnets route to an internet gateway and are where the
	// NAT gateways serving the private ones live.
	SubnetTypePublic = SubnetType{s: "public", public: true}
)

// SubnetType is whether a subnet faces the internet. A consuming casting
// filters on it: every workload it places needs a subnet, and this is the only
// fact about a subnet that both castings can predict independently.
type SubnetType struct {
	s      string
	public bool
}

func ParseSubnetType(value string) (SubnetType, error) {
	for _, subnetType := range SubnetTypes() {
		if subnetType.String() == value {
			return subnetType, nil
		}
	}

	return SubnetType{}, errors.Newf(errors.TypeInvalidInput, "failed to create subnet type from %q: it names no type", value)
}

func (subnetType SubnetType) String() string {
	return subnetType.s
}

// IsPublic reports whether the subnet routes to an internet gateway, and so
// whether a NAT gateway may be placed in it.
func (subnetType SubnetType) IsPublic() bool {
	return subnetType.public
}

func SubnetTypes() []SubnetType {
	return []SubnetType{SubnetTypePrivate, SubnetTypePublic}
}
