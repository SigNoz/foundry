package v1alpha1

import (
	"github.com/signoz/foundry/internal/types"
)

type MetaStore struct {
	// Kind of the meta store to use.
	Kind MetaStoreKind `json:"kind,omitzero" yaml:"kind,omitempty"`

	// Specification for the meta store.
	Spec MoldingSpec `json:"spec" yaml:"spec"`

	// Status of the meta store.
	Status MetaStoreStatus `json:"status" yaml:"status,omitempty"`
}

type MetaStoreStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	Addresses MetaStoreStatusAddresses `json:"addresses" yaml:"addresses,omitempty"`
}

type MetaStoreStatusAddresses struct {
	// DSN addresses.
	DSN []string `json:"dsn" yaml:"dsn"`
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
