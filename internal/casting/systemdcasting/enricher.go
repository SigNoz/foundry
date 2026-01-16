package systemdcasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*linuxMoldingEnricher)(nil)

var (
	baseTelemetryKeeperClientPort int = 9181
	baseTelemetryKeeperRaftPort   int = 9234
	baseTelemetryStoreClusterPort int = 9000
)

type linuxMoldingEnricher struct {
	materials []types.Material
}

func newLinuxMoldingEnricher(_ *v1alpha1.Casting) *linuxMoldingEnricher {
	return &linuxMoldingEnricher{materials: []types.Material{}}
}

func (enricher *linuxMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {

	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		// ClickHouse native port: 9000, HTTP port: 8123
		// For clustered setup, generate addresses for each shard/replica
		var addresses []string
		replicas := 1
		shards := 1

		cluster := config.Spec.TelemetryStore.Spec.Cluster

		if cluster.Replicas != nil {
			replicas = max(*cluster.Replicas+1, 1)
		}
		if cluster.Shards != nil {
			shards = max(*cluster.Shards, 1)
		}
		// Set CreatePerInstance flag when multiple instances on same machine
		if config.Spec.TelemetryStore.Status.Extras == nil {
			config.Spec.TelemetryStore.Status.Extras = make(map[string]string)
		}
		expectedNodes := shards * replicas
		if expectedNodes > 1 {
			// Set CreatePerInstance key to signal template to create per-instance resources
			config.Spec.TelemetryStore.Status.Extras[v1alpha1.CreatePerInstanceKey] = "true"
		}

		for shard := 0; shard < shards; shard++ {
			for replica := 0; replica < replicas; replica++ {
				// Port offset for each instance (e.g., 9000, 9001, 9002...)
				port := baseTelemetryStoreClusterPort + (shard * replicas) + replica
				addresses = append(addresses, types.FormatAddress("tcp", "localhost", port))
			}
		}
		if config.Spec.TelemetryStore.Status.Addresses == nil {
			config.Spec.TelemetryStore.Status.Addresses = make(map[string][]string)
		}
		config.Spec.TelemetryStore.Status.Addresses[v1alpha1.TelemetryStoreClusterAddresses] = addresses

	case v1alpha1.MoldingKindTelemetryKeeper:
		// ClickHouse Keeper coordination port: 9181
		var clientAddresses []string
		var raftAddresses []string

		replicas := 1
		cluster := config.Spec.TelemetryKeeper.Spec.Cluster
		if cluster.Replicas != nil {
			replicas = max(*cluster.Replicas, 1)
		}

		if config.Spec.TelemetryKeeper.Status.Extras == nil {
			config.Spec.TelemetryKeeper.Status.Extras = make(map[string]string)
		}
		// Set CreatePerInstance flag when multiple instances on same machine
		if replicas > 1 {
			config.Spec.TelemetryKeeper.Status.Extras[v1alpha1.CreatePerInstanceKey] = "true"
		}
		for replica := 0; replica < replicas; replica++ {
			clientAddresses = append(clientAddresses, types.FormatAddress("tcp", "localhost", baseTelemetryKeeperClientPort+replica))
			raftAddresses = append(raftAddresses, types.FormatAddress("tcp", "localhost", baseTelemetryKeeperRaftPort+replica))
		}

		if config.Spec.TelemetryKeeper.Status.Addresses == nil {
			config.Spec.TelemetryKeeper.Status.Addresses = make(map[string][]string)
		}

		config.Spec.TelemetryKeeper.Status.Addresses[v1alpha1.TelemetryKeeperRaftAddresses] = raftAddresses
		config.Spec.TelemetryKeeper.Status.Addresses[v1alpha1.TelemetryKeeperClientAddresses] = clientAddresses
	case v1alpha1.MoldingKindMetaStore:
		// PostgreSQL port: 5432
		if config.Spec.MetaStore.Status.Addresses == nil {
			config.Spec.MetaStore.Status.Addresses = make(map[string][]string)
		}

		password := config.Spec.MetaStore.Status.Env["POSTGRES_PASSWORD"]
		user := config.Spec.MetaStore.Status.Env["POSTGRES_USER"]
		db := config.Spec.MetaStore.Status.Env["POSTGRES_DB"]

		// Format PostgreSQL DSN: postgres://user:password@host:port/dbname?sslmode=disable
		dsn := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable", user, password, db)
		config.Spec.MetaStore.Status.Addresses[v1alpha1.MetaStoreDSNAddresses] = []string{dsn}

	case v1alpha1.MoldingKindSignoz:
		if config.Spec.Signoz.Status.Addresses == nil {
			config.Spec.Signoz.Status.Addresses = make(map[string][]string)
		}
		config.Spec.Signoz.Status.Addresses[v1alpha1.SignozAPIAddresses] = []string{
			types.FormatAddress("tcp", "localhost", 8080),
		}

	case v1alpha1.MoldingKindIngester:
		// OTel Collector gRPC: 4317, HTTP: 4318
		if config.Spec.Ingester.Status.Addresses == nil {
			config.Spec.Ingester.Status.Addresses = make(map[string][]string)
		}
		config.Spec.Ingester.Status.Addresses[v1alpha1.IngesterReceiverAddresses] = []string{
			types.FormatAddress("tcp", "localhost", 4317),
		}
	}
	return nil
}
