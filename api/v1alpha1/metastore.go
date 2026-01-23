package v1alpha1

import (
	"github.com/signoz/foundry/internal/types"
)

var (
	// MetaStoreDSNAddresses is the key for database connection addresses.
	MetaStoreDSNAddresses string = "dsn"
)

type MetaStore struct {
	// Kind of the meta store to use.
	Kind MetaStoreKind `json:"kind,omitzero" yaml:"kind,omitempty"`

	// Specification for the meta store.
	Spec MoldingSpec `json:"spec" yaml:"spec"`

	// Status of the meta store.
	Status MoldingStatus `json:"status" yaml:"status"`
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
