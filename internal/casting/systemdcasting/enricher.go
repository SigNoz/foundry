package systemdcasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*linuxMoldingEnricher)(nil)

const (
	baseTelemetryKeeperClientPort = 9181
	baseTelemetryKeeperRaftPort   = 9234
	baseTelemetryStoreClusterPort = 9000
	baseMetaStorePostgresPort     = 5432
)

type linuxMoldingEnricher struct {
	materials []types.Material
}

func newLinuxMoldingEnricher(_ *v1alpha1.Casting) *linuxMoldingEnricher {
	return &linuxMoldingEnricher{materials: []types.Material{}}
}

func (e *linuxMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
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

func (e *linuxMoldingEnricher) enrichTelemetryStore(config *v1alpha1.Casting) error {
	spec := &config.Spec.TelemetryStore
	cluster := spec.Spec.Cluster

	replicas := 1
	shards := 1
	if cluster.Replicas != nil {
		replicas = max(*cluster.Replicas+1, 1)
	}
	if cluster.Shards != nil {
		shards = max(*cluster.Shards, 1)
	}

	if replicas > 1 || shards > 1 {
		return fmt.Errorf("deployment mode '%s' does not support Distributed Clickhouse Setup, raise an issue at https://github.com/signoz/foundry/issues", config.Spec.Deployment.Mode)
	}

	// Generate addresses for each shard/replica
	var addresses []string
	for shard := 0; shard < shards; shard++ {
		for replica := 0; replica < replicas; replica++ {
			port := baseTelemetryStoreClusterPort + (shard * replicas) + replica
			addresses = append(addresses, types.FormatAddress("tcp", "localhost", port))
		}
	}

	initStatusMaps(&spec.Status)
	spec.Status.Addresses[v1alpha1.TelemetryStoreClusterAddresses] = addresses
	return nil
}

func (e *linuxMoldingEnricher) enrichTelemetryKeeper(config *v1alpha1.Casting) error {
	spec := &config.Spec.TelemetryKeeper
	cluster := spec.Spec.Cluster

	replicas := 1
	if cluster.Replicas != nil {
		replicas = max(*cluster.Replicas, 1)
	}

	if replicas > 1 {
		return fmt.Errorf("deployment mode '%s' does not support Distributed Clickhouse Setup, raise an issue at https://github.com/signoz/foundry/issues", config.Spec.Deployment.Mode)
	}

	var clientAddresses, raftAddresses []string
	for r := 0; r < replicas; r++ {
		clientAddresses = append(clientAddresses, types.FormatAddress("tcp", "localhost", baseTelemetryKeeperClientPort+r))
		raftAddresses = append(raftAddresses, types.FormatAddress("tcp", "localhost", baseTelemetryKeeperRaftPort+r))
	}

	initStatusMaps(&spec.Status)
	spec.Status.Addresses[v1alpha1.TelemetryKeeperClientAddresses] = clientAddresses
	spec.Status.Addresses[v1alpha1.TelemetryKeeperRaftAddresses] = raftAddresses
	return nil
}

func (e *linuxMoldingEnricher) enrichMetaStore(config *v1alpha1.Casting) error {
	spec := &config.Spec.MetaStore
	initStatusMaps(&spec.Status)
	dsn := types.FormatAddress("postgres", "localhost", baseMetaStorePostgresPort)
	spec.Status.Addresses[v1alpha1.MetaStoreDSNAddresses] = []string{dsn}
	return nil
}

func (e *linuxMoldingEnricher) enrichSignoz(config *v1alpha1.Casting) error {
	spec := &config.Spec.Signoz
	initStatusMaps(&spec.Status)
	spec.Status.Addresses[v1alpha1.SignozAPIAddresses] = []string{
		types.FormatAddress("tcp", "localhost", 8080),
	}
	return nil
}

func (e *linuxMoldingEnricher) enrichIngester(config *v1alpha1.Casting) error {
	spec := &config.Spec.Ingester
	initStatusMaps(&spec.Status)
	spec.Status.Addresses[v1alpha1.IngesterReceiverAddresses] = []string{
		types.FormatAddress("tcp", "localhost", 4317),
	}
	return nil
}

// initStatusMaps ensures all status maps are initialized.
func initStatusMaps(status *v1alpha1.MoldingStatus) {
	if status.Extras == nil {
		status.Extras = make(map[string]string)
	}
	if status.Addresses == nil {
		status.Addresses = make(map[string][]string)
	}
}
