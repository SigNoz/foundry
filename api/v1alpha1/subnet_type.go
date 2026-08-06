package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/jsonschema-go"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*SubnetType)(nil)
var _ yaml.Unmarshaler = (*SubnetType)(nil)
var _ json.Marshaler = (*SubnetType)(nil)
var _ json.Unmarshaler = (*SubnetType)(nil)
var _ fmt.Stringer = (*SubnetType)(nil)
var _ jsonschema.Enum = (*SubnetType)(nil)

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

func (subnetType SubnetType) MarshalJSON() ([]byte, error) {
	return json.Marshal(subnetType.String())
}

func (subnetType *SubnetType) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return subnetType.UnmarshalText([]byte(str))
}

func (subnetType *SubnetType) UnmarshalText(text []byte) error {
	for _, available := range SubnetTypes() {
		if available.String() == string(text) {
			*subnetType = available
			return nil
		}
	}

	// A nil slice is an absent value, which leaves the zero subnetType; an
	// empty string is a stated value that names none, and falls through.
	if text == nil {
		*subnetType = SubnetType{}
		return nil
	}

	return errors.New("invalid subnetType: " + string(text))
}

func (subnetType SubnetType) MarshalText() ([]byte, error) {
	return []byte(subnetType.String()), nil
}

func (subnetType *SubnetType) UnmarshalYAML(node *yaml.Node) error {
	return subnetType.UnmarshalText([]byte(node.Value))
}

func (subnetType SubnetType) MarshalYAML() (any, error) {
	return subnetType.String(), nil
}

func (subnetType SubnetType) Enum() []any {
	subnetTypes := []any{}
	for _, subnetType := range SubnetTypes() {
		subnetTypes = append(subnetTypes, subnetType.String())
	}

	return subnetTypes
}
