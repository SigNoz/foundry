package v1alpha1

import (
	"encoding/json"
	"errors"

	"github.com/signoz/foundry/internal/types"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*MetaStoreKind)(nil)
var _ yaml.Unmarshaler = (*MetaStoreKind)(nil)

var _ json.Marshaler = (*MetaStoreKind)(nil)
var _ json.Unmarshaler = (*MetaStoreKind)(nil)

var (
	MetaStoreKindPostgres MetaStoreKind = MetaStoreKind{s: "postgres"}
	MetaStoreKindSQLite   MetaStoreKind = MetaStoreKind{s: "sqlite"}
)

var (
	// MetaStoreDSNAddresses is the key for database connection addresses.
	MetaStoreDSNAddresses string = "dsn"
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

func (kind MetaStoreKind) MarshalYAML() (interface{}, error) {
	return kind.String(), nil
}

type MetaStore struct {
	// Kind of the meta store to use.
	Kind MetaStoreKind `json:"kind,omitzero" yaml:"kind,omitempty"`

	// Specification for the meta store.
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	// Status of the meta store.
	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func DefaultMetaStore() MetaStore {
	return MetaStore{
		Kind: MetaStoreKindPostgres,
		Spec: MoldingSpec{
			Enabled: true,
			Cluster: TypeCluster{
				Replicas: types.NewIntPtr(1),
			},
			Version: "16",
			Image:   "postgres:16",
		},
	}
}
