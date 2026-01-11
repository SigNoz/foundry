package telemetrykeepermolding

import (
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

func Default() *v1alpha1.TelemetryKeeper {
	return &v1alpha1.TelemetryKeeper{
		Kind: v1alpha1.TelemetryKeeperKindClickhouseKeeper,
		Spec: v1alpha1.MoldingSpec{
			Enabled: true,
			Cluster: v1alpha1.TypeCluster{
				Replicas: types.NewIntPtr(1),
			},
			Version: "25.5.6",
			Image:   "clickhouse/clickhouse-keeper:25.5.6",
			Env:     map[string]string{},
		},
	}
}
