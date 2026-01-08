package rendercasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*renderMoldingEnricher)(nil)

type renderMoldingEnricher struct {
	material types.Material
}

func newRenderMoldingEnricher(config *v1alpha1.Casting) *renderMoldingEnricher {
	return &renderMoldingEnricher{material: types.Material{}}
}

func (enricher *renderMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		// For Render, we use service names for addresses
		// Service names follow format: name-telemetrystore-kind-shard-replica
		// Collect all addresses from all shards and replicas
		var addresses []string
		replicas := 1
		shards := 1
		cluster := config.Spec.TelemetryKeeper.Spec.Cluster
		if cluster.Replicas != nil {
			replicas = max(*config.Spec.TelemetryStore.Spec.Cluster.Replicas+1, 1)
		}
		if cluster.Shards != nil {
			shards = max(*config.Spec.TelemetryStore.Spec.Cluster.Shards, 1)
		}
		for shardIdx := 0; shardIdx < shards; shardIdx++ {
			for replicaIdx := 0; replicaIdx < replicas; replicaIdx++ {
				serviceName := fmt.Sprintf("%s-telemetrystore-%s-%d-%d", config.Metadata.Name, config.Spec.TelemetryStore.Kind.String(), shardIdx, replicaIdx)
				address := types.FormatAddress("tcp", serviceName, 9000)
				addresses = append(addresses, address)
			}
		}
		config.Spec.TelemetryStore.Status.Addresses.TCP = addresses

	case v1alpha1.MoldingKindSignoz:
		// For Render, we use service names for addresses
		// Service names follow format: name-signoz-N
		serviceName := fmt.Sprintf("%s-signoz", config.Metadata.Name)
		address := types.FormatAddress("http", serviceName, 8080)
		config.Spec.Signoz.Status.Addresses.APIServer = []string{address}
		config.Spec.Signoz.Status.Addresses.Opamp = []string{address}

	case v1alpha1.MoldingKindTelemetryKeeper:
		// For Render, we use service names for addresses
		// Service names follow format: name-telemetrykeeper-kind-N
		// We need to collect all replica addresses
		var clientAddresses []string
		var raftAddresses []string
		replicas := int(*config.Spec.TelemetryKeeper.Spec.Cluster.Replicas)
		for i := 0; i < replicas; i++ {
			serviceName := fmt.Sprintf("%s-telemetrykeeper-%s-%d", config.Metadata.Name, config.Spec.TelemetryKeeper.Kind.String(), i)
			clientAddress := types.FormatAddress("tcp", serviceName, 9181)
			raftAddress := types.FormatAddress("tcp", serviceName, 9234)
			clientAddresses = append(clientAddresses, clientAddress)
			raftAddresses = append(raftAddresses, raftAddress)
		}
		config.Spec.TelemetryKeeper.Status.Addresses.Client = clientAddresses
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = raftAddresses

	case v1alpha1.MoldingKindIngester:
		// For Render, we use service names for addresses
		// Service names follow format: name-ingester-N
		var addresses []string
		replicas := int(*config.Spec.Ingester.Spec.Cluster.Replicas)
		for i := 0; i < replicas; i++ {
			serviceName := fmt.Sprintf("%s-ingester-%d", config.Metadata.Name, i)
			address := types.FormatAddress("tcp", serviceName, 4318)
			addresses = append(addresses, address)
		}
		config.Spec.Ingester.Status.Addresses.OTLP = addresses
	}

	return nil
}
