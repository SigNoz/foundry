package v1alpha1

import (
	"errors"

	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*MetaStoreKind)(nil)
var _ yaml.Unmarshaler = (*MetaStoreKind)(nil)

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

func (kind *MetaStoreKind) UnmarshalText(text []byte) error {
	for _, availableKind := range MetaStoreKinds() {
		if availableKind.String() == string(text) {
			*kind = availableKind
			return nil
		}
	}
	return errors.New("invalid meta store kind: " + string(text))
}

func (kind MetaStoreKind) MarshalText() ([]byte, error) {
	return []byte(kind.String()), nil
}

func (kind *MetaStoreKind) UnmarshalYAML(node *yaml.Node) error {
	return kind.UnmarshalText([]byte(node.Value))
}

func (kind MetaStoreKind) MarshalYAML() (interface{}, error) {
	return kind.String(), nil
}

type MetaStore struct {
	// Kind of the meta store to use.
	Kind MetaStoreKind `json:"kind,omitempty" yaml:"kind,omitempty"`

	// Specification for the meta store.
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	// Status of the meta store.
	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}
