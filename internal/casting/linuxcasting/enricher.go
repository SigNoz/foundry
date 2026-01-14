package linuxcasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*linuxMoldingEnricher)(nil)

type linuxMoldingEnricher struct {
	materials []types.Material
}

func newLinuxMoldingEnricher(config *v1alpha1.Casting) (*linuxMoldingEnricher, error) {
	materials, err := getServiceMaterials(config)
	if err != nil {
		return nil, err
	}

	return &linuxMoldingEnricher{materials: materials}, nil
}

func (enricher *linuxMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {

	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		// ClickHouse native port: 9000, HTTP port: 8123
		// For clustered setup, generate addresses for each shard/replica

		var addresses []string

		cluster := config.Spec.TelemetryStore.Spec.Cluster
		replicas := max(*cluster.Replicas, 1)
		shards := max(*cluster.Shards, 1)
	
		for shard := 0; shard < shards; shard++ {
			for replica := 0; replica < replicas; replica++ {
				// Port offset for each instance (e.g., 9000, 9001, 9002...)
				port := 9000 + (shard * replicas) + replica
				addresses = append(addresses, types.FormatAddress("tcp", "localhost", port))
			}
		}
		config.Spec.TelemetryStore.Status.Addresses = addresses

	case v1alpha1.MoldingKindTelemetryKeeper:
		// ClickHouse Keeper coordination port: 9181
		var addresses []string
		replicas := 1
		if config.Spec.TelemetryKeeper.Spec.Cluster.Replicas != nil {
			replicas = *config.Spec.TelemetryKeeper.Spec.Cluster.Replicas
		}

		for replica := 0; replica < replicas; replica++ {
			port := 9181 + replica
			addresses = append(addresses, types.FormatAddress("tcp", "localhost", port))
		}
		config.Spec.TelemetryKeeper.Status.Addresses = addresses

	case v1alpha1.MoldingKindMetaStore:
		// PostgreSQL port: 5432
		config.Spec.MetaStore.Status.Addresses = []string{
			types.FormatAddress("tcp", "localhost", 5432),
		}

	case v1alpha1.MoldingKindSignoz:
		config.Spec.Signoz.Status.Addresses = []string{
			types.FormatAddress("tcp", "localhost", 8080),
		}

	case v1alpha1.MoldingKindIngester:
		// OTel Collector gRPC: 4317, HTTP: 4318
		config.Spec.Ingester.Status.Addresses = []string{
			types.FormatAddress("tcp", "localhost", 4317),
		}
	}
	return nil
}
