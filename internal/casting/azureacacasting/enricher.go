package azureacacasting

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
	metaStorePostgresPort     = 5432
	signozOpampPort           = 4320
)

var _ molding.MoldingEnricher = (*acaMoldingEnricher)(nil)

type acaMoldingEnricher struct{}

func newAcaMoldingEnricher(_ *v1alpha1.Casting) *acaMoldingEnricher {
	return &acaMoldingEnricher{}
}

func (e *acaMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
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

func (e *acaMoldingEnricher) enrichTelemetryStore(config *v1alpha1.Casting) error {
	// In ACA, container apps reference each other by name within the environment.
	name := config.Metadata.Name + "-clickhouse"
	config.Spec.TelemetryStore.Status.Addresses.TCP = []string{types.FormatAddress("tcp", name, telemetryStorePort)}
	return nil
}

func (e *acaMoldingEnricher) enrichTelemetryKeeper(config *v1alpha1.Casting) error {
	spec := &config.Spec.TelemetryKeeper
	replicas := 1
	if spec.Spec.Cluster.Replicas != nil && *spec.Spec.Cluster.Replicas > 0 {
		replicas = *spec.Spec.Cluster.Replicas
	}
	base := config.Metadata.Name + "-clickhouse-keeper"
	var client, raft []string
	for i := 0; i < replicas; i++ {
		name := fmt.Sprintf("%s-%d", base, i)
		client = append(client, types.FormatAddress("tcp", name, telemetryKeeperClientPort))
		raft = append(raft, types.FormatAddress("tcp", name, telemetryKeeperRaftPort))
	}
	config.Spec.TelemetryKeeper.Status.Addresses.Client = client
	config.Spec.TelemetryKeeper.Status.Addresses.Raft = raft
	return nil
}

func (e *acaMoldingEnricher) enrichMetaStore(config *v1alpha1.Casting) error {
	name := config.Metadata.Name + "-metastore-postgres"
	config.Spec.MetaStore.Status.Addresses.DSN = []string{types.FormatAddress("tcp", name, metaStorePostgresPort)}
	return nil
}

func (e *acaMoldingEnricher) enrichSignoz(config *v1alpha1.Casting) error {
	name := config.Metadata.Name + "-signoz"
	config.Spec.Signoz.Status.Addresses.Opamp = []string{types.FormatAddress("ws", name, signozOpampPort)}
	return nil
}

func (e *acaMoldingEnricher) enrichIngester(config *v1alpha1.Casting) error {
	// No-op: ingester molding derives from other statuses.
	return nil
}
