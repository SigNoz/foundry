package collectionagent

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/jsonschema-go"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*CollectorKind)(nil)
var _ yaml.Unmarshaler = (*CollectorKind)(nil)
var _ json.Marshaler = (*CollectorKind)(nil)
var _ json.Unmarshaler = (*CollectorKind)(nil)
var _ fmt.Stringer = (*CollectorKind)(nil)
var _ jsonschema.Enum = (*CollectorKind)(nil)

var (
	CollectorKindAgent CollectorKind = CollectorKind{s: "agent"}
)

type CollectorKind struct {
	s string
}

func (kind CollectorKind) String() string {
	return kind.s
}

func CollectorKinds() []CollectorKind {
	return []CollectorKind{CollectorKindAgent}
}

func (kind CollectorKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(kind.String())
}

func (kind *CollectorKind) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return kind.UnmarshalText([]byte(str))
}

func (kind *CollectorKind) UnmarshalText(text []byte) error {
	for _, availableKind := range CollectorKinds() {
		if availableKind.String() == string(text) {
			*kind = availableKind
			return nil
		}
	}
	if text == nil {
		*kind = CollectorKind{s: ""}
		return nil
	}
	return errors.New("invalid collector kind: " + string(text))
}

func (kind CollectorKind) MarshalText() ([]byte, error) {
	return []byte(kind.String()), nil
}

func (kind *CollectorKind) UnmarshalYAML(node *yaml.Node) error {
	return kind.UnmarshalText([]byte(node.Value))
}

func (kind CollectorKind) MarshalYAML() (any, error) {
	return kind.String(), nil
}

func (kind CollectorKind) Enum() []any {
	kinds := []any{}
	for _, kind := range CollectorKinds() {
		kinds = append(kinds, kind.String())
	}

	return kinds
}
