package kuberneteskustomizecasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

const (
	telemetryStorePort        = 9000
	telemetryKeeperClientPort = 9181
	telemetryKeeperRaftPort   = 9234
	signozOpampPort           = 4320
)

var _ molding.MoldingEnricher = (*kustomizeMoldingEnricher)(nil)

type kustomizeMoldingEnricher struct {
	materials []types.Material
}

func newKustomizeMoldingEnricher(_ *v1alpha1.Casting) *kustomizeMoldingEnricher {
	return &kustomizeMoldingEnricher{materials: []types.Material{}}
}

func (e *kustomizeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		return e.enrichTelemetryStore(config)
	case v1alpha1.MoldingKindTelemetryKeeper:
		return e.enrichTelemetryKeeper(config)
	case v1alpha1.MoldingKindMetaStore:
		return e.enrichMetaStore(config)
	case v1alpha1.MoldingKindSignoz:
		return e.enrichSignoz(config)
	case v1alpha1.MoldingKindIngester:
		return e.enrichIngester(config)
	}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichTelemetryStore(config *v1alpha1.Casting) error {
	name := config.Metadata.Name + "-clickhouse"
	config.Spec.TelemetryStore.Status.Addresses.TCP = []string{types.FormatAddress("tcp", name, telemetryStorePort)}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichTelemetryKeeper(config *v1alpha1.Casting) error {
	spec := &config.Spec.TelemetryKeeper
	replicas := 1
	if spec.Spec.Cluster.Replicas != nil && *spec.Spec.Cluster.Replicas > 0 {
		replicas = *spec.Spec.Cluster.Replicas
	}
	if replicas < 1 {
		replicas = 1
	}
	base := config.Metadata.Name + "-clickhouse-keeper"
	var client, raft []string
	for i := 0; i < replicas; i++ {
		client = append(client, types.FormatAddress("tcp", fmt.Sprintf("%s-%d", base, i), telemetryKeeperClientPort))
		raft = append(raft, types.FormatAddress("tcp", fmt.Sprintf("%s-%d", base, i), telemetryKeeperRaftPort))
	}
	config.Spec.TelemetryKeeper.Status.Addresses.Client = client
	config.Spec.TelemetryKeeper.Status.Addresses.Raft = raft
	return nil
}

func (e *kustomizeMoldingEnricher) enrichMetaStore(config *v1alpha1.Casting) error {
	name := config.Metadata.Name + "-metastore"
	config.Spec.MetaStore.Status.Addresses.DSN = []string{
		fmt.Sprintf("postgres://%s:5432", name),
	}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichSignoz(config *v1alpha1.Casting) error {
	name := config.Metadata.Name + "-signoz"
	config.Spec.Signoz.Status.Addresses.Opamp = []string{types.FormatAddress("tcp", name, signozOpampPort)}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichIngester(config *v1alpha1.Casting) error {
	// No-op: ingester molding only writes Status.Config.Data from other status.
	return nil
}
