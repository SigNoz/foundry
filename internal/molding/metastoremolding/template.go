package metastoremolding

import (
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

func Default() *v1alpha1.MetaStore {
	return &v1alpha1.MetaStore{
		Kind: v1alpha1.MetaStoreKindPostgres,
		Spec: v1alpha1.MoldingSpec{
			Enabled: true,
			Cluster: v1alpha1.TypeCluster{
				Replicas: types.NewIntPtr(1),
			},
			Version: "16",
			Image:   "postgres:16",
			Env: map[string]string{
				"POSTGRES_USER":     "signoz",
				"POSTGRES_PASSWORD": "password",
				"POSTGRES_DB":       "signoz",
			},
		},
	}
}
