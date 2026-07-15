package infrastructure

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/swaggest/jsonschema-go"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*ResourceKind)(nil)
var _ yaml.Unmarshaler = (*ResourceKind)(nil)
var _ json.Marshaler = (*ResourceKind)(nil)
var _ json.Unmarshaler = (*ResourceKind)(nil)
var _ fmt.Stringer = (*ResourceKind)(nil)
var _ jsonschema.Enum = (*ResourceKind)(nil)

var (
	ResourceKindInstallation    ResourceKind = ResourceKind{s: v1alpha1.KindInstallation.String()}
	ResourceKindCollectionAgent ResourceKind = ResourceKind{s: v1alpha1.KindCollectionAgent.String()}
)

type ResourceKind struct {
	s string
}

func (kind ResourceKind) String() string {
	return kind.s
}

func ResourceKinds() []ResourceKind {
	return []ResourceKind{ResourceKindInstallation, ResourceKindCollectionAgent}
}

func (kind ResourceKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(kind.String())
}

func (kind *ResourceKind) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return kind.UnmarshalText([]byte(str))
}

func (kind *ResourceKind) UnmarshalText(text []byte) error {
	for _, availableKind := range ResourceKinds() {
		if availableKind.String() == string(text) {
			*kind = availableKind
			return nil
		}
	}
	if text == nil {
		*kind = ResourceKind{s: ""}
		return nil
	}
	return errors.New("invalid resource kind: " + string(text))
}

func (kind ResourceKind) MarshalText() ([]byte, error) {
	return []byte(kind.String()), nil
}

func (kind *ResourceKind) UnmarshalYAML(node *yaml.Node) error {
	return kind.UnmarshalText([]byte(node.Value))
}

func (kind ResourceKind) MarshalYAML() (any, error) {
	return kind.String(), nil
}

func (kind ResourceKind) Enum() []any {
	kinds := []any{}
	for _, kind := range ResourceKinds() {
		kinds = append(kinds, kind.String())
	}

	return kinds
}
