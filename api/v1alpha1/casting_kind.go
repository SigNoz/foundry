package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/jsonschema-go"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*Kind)(nil)
var _ yaml.Unmarshaler = (*Kind)(nil)
var _ json.Marshaler = (*Kind)(nil)
var _ json.Unmarshaler = (*Kind)(nil)
var _ fmt.Stringer = (*Kind)(nil)
var _ jsonschema.Enum = (*Kind)(nil)

var (
	KindInstallation Kind = Kind{s: "Installation"}
)

// Kind discriminates between top-level casting resource types.
// An empty/missing kind unmarshals to KindInstallation for backwards compatibility
// with casting files written before kind was introduced.
type Kind struct {
	s string
}

func (kind Kind) String() string {
	return kind.s
}

func Kinds() []Kind {
	return []Kind{KindInstallation}
}

func (kind Kind) MarshalJSON() ([]byte, error) {
	return json.Marshal(kind.String())
}

func (kind *Kind) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return kind.UnmarshalText([]byte(str))
}

func (kind *Kind) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*kind = KindInstallation
		return nil
	}

	for _, available := range Kinds() {
		if available.String() == string(text) {
			*kind = available
			return nil
		}
	}

	return errors.New("invalid kind: " + string(text))
}

func (kind Kind) MarshalText() ([]byte, error) {
	return []byte(kind.String()), nil
}

func (kind *Kind) UnmarshalYAML(node *yaml.Node) error {
	return kind.UnmarshalText([]byte(node.Value))
}

func (kind Kind) MarshalYAML() (any, error) {
	return kind.String(), nil
}

func (kind Kind) Enum() []any {
	kinds := []any{}
	for _, kind := range Kinds() {
		kinds = append(kinds, kind.String())
	}

	return kinds
}
