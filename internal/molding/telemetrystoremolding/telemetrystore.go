package telemetrystoremolding

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
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

func (molding *telemetrystore) MoldV1Alpha1(ctx context.Context, config *installation.Casting) error {
	data, err := molding.getData(config)
	if err != nil {
		molding.logger.ErrorContext(ctx, "failed to get data", foundryerrors.LogAttr(err))
		return err
	}

	// Extract enricher config overrides (applies to all nodes).
	overrides := config.Spec.TelemetryStore.Status.Extras["_overrides"]

	functionBuf := bytes.NewBuffer(nil)
	if err := FunctionsClickhousev2556YAML.Execute(functionBuf, data); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute functions template")
	}

	// Generate per-node configs (each node needs its own macros.shard / macros.replica).
	configs := make(map[string]string, data.ShardCount*data.ReplicaCount+1)
	for s := 0; s < data.ShardCount; s++ {
		for r := 0; r < data.ReplicaCount; r++ {
			data.ShardID = s
			data.ReplicaID = r

			configBuf := bytes.NewBuffer(nil)
			if err := ConfigClickhousev2556YAML.Execute(configBuf, data); err != nil {
				return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute config template for shard %d replica %d", s, r)
			}

			key := fmt.Sprintf("config-%d-%d.yaml", s, r)
			base := configBuf.String()

			if overrides != "" {
				merged, err := domain.MergeYAML(base, overrides)
				if err != nil {
					return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to merge config overrides for %s", key)
				}
				base = merged
			}

			configs[key] = base
		}
	}
	configs["functions.yaml"] = functionBuf.String()

	config.Spec.TelemetryStore.Status.Config.Data = configs

	return nil
}

func (molding *telemetrystore) getData(config *installation.Casting) (Data, error) {
	storeAddresses := config.Spec.TelemetryStore.Status.Addresses.TCP
	if len(storeAddresses) == 0 {
		return Data{}, foundryerrors.Newf(foundryerrors.TypeInternal, "telemetry store addresses not set in status")
	}

	cluster := config.Spec.TelemetryStore.Spec.Cluster

	shardCount := 1
	if cluster.Shards != nil && *cluster.Shards > 0 {
		shardCount = *cluster.Shards
	}

	replicaCount := 1
	if cluster.Replicas != nil {
		replicaCount = *cluster.Replicas + 1
	}

	expectedNodes := shardCount * replicaCount
	if len(storeAddresses) < expectedNodes {
		return Data{}, foundryerrors.Newf(
			foundryerrors.TypeInvalidInput,
			"insufficient addresses: have %d, need %d (shards=%d × replicas=%d)",
			len(storeAddresses), expectedNodes, shardCount, replicaCount,
		)
	}

	newStoreAddrs, err := domain.ParseAddresses(storeAddresses[:expectedNodes])
	if err != nil {
		return Data{}, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to parse addresses")
	}

	keeperAddresses := config.Spec.TelemetryKeeper.Status.Addresses.Client
	if len(keeperAddresses) == 0 {
		return Data{}, foundryerrors.Newf(foundryerrors.TypeInternal, "telemetry keeper addresses not set in status")
	}

	newKeeperAddrs, err := domain.ParseAddresses(keeperAddresses)
	if err != nil {
		return Data{}, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to parse addresses")
	}

	return Data{
		StoreAddresses:  newStoreAddrs,
		KeeperAddresses: newKeeperAddrs,
		ShardCount:      shardCount,
		ReplicaCount:    replicaCount,
	}, nil
}
