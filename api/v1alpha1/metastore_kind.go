package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*MetaStoreKind)(nil)
var _ yaml.Unmarshaler = (*MetaStoreKind)(nil)
var _ json.Marshaler = (*MetaStoreKind)(nil)
var _ json.Unmarshaler = (*MetaStoreKind)(nil)
var _ fmt.Stringer = (*MetaStoreKind)(nil)

var (
	MetaStoreKindPostgres MetaStoreKind = MetaStoreKind{s: "postgres"}
	MetaStoreKindSQLite   MetaStoreKind = MetaStoreKind{s: "sqlite"}
)

type MetaStoreKind struct {
	s string
}

func (kind MetaStoreKind) String() string {
	return kind.s
}

func MetaStoreKinds() []MetaStoreKind {
	return []MetaStoreKind{MetaStoreKindPostgres, MetaStoreKindSQLite}
}

func (kind MetaStoreKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(kind.String())
}

func (kind *MetaStoreKind) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return kind.UnmarshalText([]byte(str))
}

func (kind *MetaStoreKind) UnmarshalText(text []byte) error {
	for _, availableKind := range MetaStoreKinds() {
		if availableKind.String() == string(text) {
			*kind = availableKind
			return nil
		}
	}
	if text == nil {
		*kind = MetaStoreKind{s: ""}
		return nil
	}
	return errors.New("invalid meta store kind: " + string(text))
}

func (kind MetaStoreKind) MarshalText() ([]byte, error) {
	return []byte(kind.String()), nil
}

func (kind *MetaStoreKind) UnmarshalYAML(node *yaml.Node) error {
	return kind.UnmarshalText([]byte(node.Value))
}

func (kind MetaStoreKind) MarshalYAML() (any, error) {
	return kind.String(), nil
}
