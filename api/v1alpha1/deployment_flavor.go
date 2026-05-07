package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/jsonschema-go"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*Flavor)(nil)
var _ yaml.Unmarshaler = (*Flavor)(nil)
var _ json.Marshaler = (*Flavor)(nil)
var _ json.Unmarshaler = (*Flavor)(nil)
var _ fmt.Stringer = (*Flavor)(nil)
var _ jsonschema.Enum = (*Flavor)(nil)

var (
	FlavorCompose   Flavor = Flavor{s: "compose"}
	FlavorSwarm     Flavor = Flavor{s: "swarm"}
	FlavorBinary    Flavor = Flavor{s: "binary"}
	FlavorKustomize Flavor = Flavor{s: "kustomize"}
	FlavorHelm      Flavor = Flavor{s: "helm"}
	FlavorBlueprint Flavor = Flavor{s: "blueprint"}
	FlavorStack     Flavor = Flavor{s: "stack"}
	FlavorTemplate  Flavor = Flavor{s: "template"}
	FlavorTerraform Flavor = Flavor{s: "terraform"}
)

type Flavor struct {
	s string
}

func (flavor Flavor) String() string {
	return flavor.s
}

func Flavors() []Flavor {
	return []Flavor{
		FlavorCompose,
		FlavorSwarm,
		FlavorBinary,
		FlavorKustomize,
		FlavorHelm,
		FlavorBlueprint,
		FlavorStack,
		FlavorTemplate,
		FlavorTerraform,
	}
}

func (flavor Flavor) MarshalJSON() ([]byte, error) {
	return json.Marshal(flavor.String())
}

func (flavor *Flavor) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return flavor.UnmarshalText([]byte(str))
}

func (flavor *Flavor) UnmarshalText(text []byte) error {
	for _, available := range Flavors() {
		if available.String() == string(text) {
			*flavor = available
			return nil
		}
	}

	if len(text) == 0 {
		*flavor = Flavor{s: ""}
		return nil
	}

	return errors.New("invalid deployment flavor: " + string(text))
}

func (flavor Flavor) MarshalText() ([]byte, error) {
	return []byte(flavor.String()), nil
}

func (flavor *Flavor) UnmarshalYAML(node *yaml.Node) error {
	return flavor.UnmarshalText([]byte(node.Value))
}

func (flavor Flavor) MarshalYAML() (any, error) {
	return flavor.String(), nil
}

func (flavor Flavor) Enum() []any {
	flavors := []any{}
	for _, flavor := range Flavors() {
		flavors = append(flavors, flavor.String())
	}

	return flavors
}
