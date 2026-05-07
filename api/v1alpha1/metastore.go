package v1alpha1

type MetaStore struct {
	// Kind of the meta store to use.
	Kind MetaStoreKind `json:"kind,omitzero" yaml:"kind,omitempty" description:"Kind of the meta store to use" examples:"[\"postgres\",\"sqlite\"]"`

	// Specification for the meta store.
	Spec MoldingSpec `json:"spec" yaml:"spec" description:"Specification for the meta store"`

	// Status of the meta store.
	Status MetaStoreStatus `json:"status" yaml:"status,omitempty" description:"Status of the meta store"`

	_ struct{} `additionalProperties:"false"`
}

type MetaStoreStatus struct {
	MoldingStatus `json:",inline" yaml:",inline"`

	Addresses MetaStoreStatusAddresses `json:"addresses" yaml:"addresses,omitempty" description:"Addresses of the meta store"`

	_ struct{} `additionalProperties:"false"`
}

type MetaStoreStatusAddresses struct {
	// DSN addresses.
	DSN []string `json:"dsn" yaml:"dsn" description:"DSN addresses"`
	_   struct{} `additionalProperties:"false"`
}

func DefaultMetaStore() MetaStore {
	return MetaStore{
		Kind: MetaStoreKindPostgres,
		Spec: MoldingSpec{
			Enabled: boolPtr(true),
			Cluster: TypeCluster{
				Replicas: intPtr(1),
			},
			Version: "16",
			Image:   "postgres:16",
		},
	}
}
