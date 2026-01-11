package signozmolding

import (
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

func Default() *v1alpha1.SigNoz {
	return &v1alpha1.SigNoz{
		Spec: v1alpha1.MoldingSpec{
			Enabled: true,
			Cluster: v1alpha1.TypeCluster{
				Replicas: types.NewIntPtr(1),
			},
			Version: "latest",
			Image:   "signoz/signoz:latest",
			Env: map[string]string{
				"SIGNOZ_ALERTMANAGER_PROVIDER":   "signoz",
				"SIGNOZ_TELEMETRYSTORE_PROVIDER": "clickhouse",
				"SIGNOZ_SQLSTORE_PROVIDER":       "postgres",
			},
		},
	}
}
