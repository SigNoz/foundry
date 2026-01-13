package telemetrystoremolding

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

const (
	defaultShardCount   = 1
	defaultReplicaCount = 1
)

var _ molding.Molding = (*telemetrystore)(nil)

type telemetrystore struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *telemetrystore {
	return &telemetrystore{
		logger: logger,
	}
}

func (molding *telemetrystore) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindTelemetryStore
}

func (molding *telemetrystore) MoldV1Alpha1(ctx context.Context, config *v1alpha1.Casting) error {
	data, err := molding.getData(config)
	if err != nil {
		molding.logger.ErrorContext(ctx, "failed to get data", foundryerrors.LogAttr(err))
		return err
	}

	configBuf := bytes.NewBuffer(nil)
	if err := ConfigClickhousev2556YAML.Execute(configBuf, data); err != nil {
		return fmt.Errorf("failed to execute config template: %w", err)
	}

	config.Spec.TelemetryStore.Spec.Config.Data = map[string]string{
		"config.yaml": configBuf.String(),
	}

	return nil
}

func (molding *telemetrystore) getData(config *v1alpha1.Casting) (Data, error) {
	addresses := config.Spec.TelemetryStore.Status.Addresses
	if len(addresses) == 0 {
		return Data{}, fmt.Errorf("telemetry store addresses not set in status")
	}

	clusterConfig, err := molding.getTelemetryStoreCluster(config.Spec.TelemetryStore.Spec.Cluster, addresses)
	if err != nil {
		return Data{}, fmt.Errorf("failed to build cluster config: %w", err)
	}

	return Data{
		TelemetryStoreClickHouseCluster: clusterConfig,
	}, nil
}

func (molding *telemetrystore) getTelemetryStoreCluster(cluster v1alpha1.TypeCluster, addresses []string) (ClusterConfig, error) {
	shardCount := defaultShardCount
	replicaCount := defaultReplicaCount

	if cluster.Shards != nil {
		shardCount = *cluster.Shards
	}
	if cluster.Replicas != nil {
		replicaCount = *cluster.Replicas
	}

	expectedNodes := shardCount * replicaCount
	if len(addresses) < expectedNodes {
		return ClusterConfig{}, fmt.Errorf(
			"insufficient addresses: have %d, need %d (shards=%d × replicas=%d)",
			len(addresses), expectedNodes, shardCount, replicaCount,
		)
	}

	parsedAddrs, err := types.ParseAddresses(addresses[:expectedNodes])
	if err != nil {
		return ClusterConfig{}, fmt.Errorf("failed to parse addresses: %w", err)
	}

	shards := make([]ShardConfig, 0, shardCount)
	addrIdx := 0

	for s := 0; s < shardCount; s++ {
		replicas := make([]ReplicaConfig, 0, replicaCount)
		for r := 0; r < replicaCount; r++ {
			addr := parsedAddrs[addrIdx]
			replicas = append(replicas, ReplicaConfig{
				Host: addr.Host,
				Port: addr.Port,
			})
			addrIdx++
		}
		shards = append(shards, ShardConfig{Replicas: replicas})
	}

	return ClusterConfig{Shards: shards}, nil
}
