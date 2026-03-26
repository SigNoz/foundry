package infrastructure

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*ComputeType)(nil)
var _ yaml.Unmarshaler = (*ComputeType)(nil)
var _ json.Marshaler = (*ComputeType)(nil)
var _ json.Unmarshaler = (*ComputeType)(nil)
var _ fmt.Stringer = (*ComputeType)(nil)

var (
	// AWS compute types.
	ComputeTypeEC2 ComputeType = ComputeType{s: "ec2"}
	ComputeTypeEKS ComputeType = ComputeType{s: "eks"}
	// GCP compute types.
	ComputeTypeGCE ComputeType = ComputeType{s: "gce"}
	ComputeTypeGKE ComputeType = ComputeType{s: "gke"}
	// Azure compute types.
	ComputeTypeVM  ComputeType = ComputeType{s: "vm"}
	ComputeTypeAKS ComputeType = ComputeType{s: "aks"}
)

// ComputeType identifies the compute resource type for a given cloud provider.
// It is an internal type resolved from the provider + deployment combination —
// users do not set this directly.
type ComputeType struct {
	s string
}

func (c ComputeType) String() string {
	return c.s
}

func (c ComputeType) IsZero() bool {
	return c.s == ""
}

func ComputeTypes() []ComputeType {
	return []ComputeType{
		ComputeTypeEC2,
		ComputeTypeEKS,
		ComputeTypeGCE,
		ComputeTypeGKE,
		ComputeTypeVM,
		ComputeTypeAKS,
	}
}

func (c ComputeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *ComputeType) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}
	return c.UnmarshalText([]byte(str))
}

func (c *ComputeType) UnmarshalText(text []byte) error {
	for _, available := range ComputeTypes() {
		if available.String() == string(text) {
			*c = available
			return nil
		}
	}
	if len(text) == 0 {
		*c = ComputeType{s: ""}
		return nil
	}
	return errors.New("invalid infrastructure compute type: " + string(text))
}

func (c ComputeType) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *ComputeType) UnmarshalYAML(node *yaml.Node) error {
	return c.UnmarshalText([]byte(node.Value))
}

func (c ComputeType) MarshalYAML() (any, error) {
	return c.String(), nil
}
