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

	config.Spec.TelemetryStore.Status.Addresses.TCP = addresses
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

	config.Spec.TelemetryKeeper.Status.Addresses.Client = clientAddresses
	config.Spec.TelemetryKeeper.Status.Addresses.Raft = raftAddresses
	return nil
}

func (e *linuxMoldingEnricher) enrichMetaStore(config *v1alpha1.Casting) error {
	dsn := types.FormatAddress("postgres", "localhost", baseMetaStorePostgresPort)
	config.Spec.MetaStore.Status.Addresses.DSN = []string{dsn}

	// Get the annotation value
	metastoreBin := config.Metadata.Annotations["foundry.signoz.io/metastore-postgres-binary-path"]

	// If it's missing, apply the default and write it back
	if metastoreBin == "" {
		metastoreBin = "/usr/bin/postgres"

		if config.Metadata.Annotations == nil {
			config.Metadata.Annotations = make(map[string]string)
		}
		config.Metadata.Annotations["foundry.signoz.io/metastore-postgres-binary-path"] = metastoreBin
	}
	return nil
}

func (e *linuxMoldingEnricher) enrichSignoz(config *v1alpha1.Casting) error {
	config.Spec.Signoz.Status.Addresses.Opamp = []string{
		types.FormatAddress("ws", "localhost", 4320),
	}
	config.Spec.Signoz.Status.Addresses.APIServer = []string{
		types.FormatAddress("tcp", "localhost", 8080),
	}

	// Get the annotation value
	signozBin := config.Metadata.Annotations["foundry.signoz.io/signoz-binary-path"]

	// If it's missing, apply the default and write it back
	if signozBin == "" {
		signozBin = "/opt/signoz/bin/signoz"

		if config.Metadata.Annotations == nil {
			config.Metadata.Annotations = make(map[string]string)
		}
		config.Metadata.Annotations["foundry.signoz.io/signoz-binary-path"] = signozBin
	}

	return nil
}

func (e *linuxMoldingEnricher) enrichIngester(config *v1alpha1.Casting) error {
	config.Spec.Ingester.Status.Addresses.OTLP = []string{
		types.FormatAddress("tcp", "localhost", 4317),
	}

	// Get the annotation value
	ingesterBin := config.Metadata.Annotations["foundry.signoz.io/ingester-binary-path"]

	// If it's missing, apply the default and write it back
	if ingesterBin == "" {
		ingesterBin = "/opt/ingester/bin/signoz-otel-collector"

		if config.Metadata.Annotations == nil {
			config.Metadata.Annotations = make(map[string]string)
		}
		config.Metadata.Annotations["foundry.signoz.io/ingester-binary-path"] = ingesterBin
	}

	return nil
}
