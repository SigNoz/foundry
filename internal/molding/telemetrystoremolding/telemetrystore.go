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

	// Extract enricher config overrides (applies to all nodes).
	overrides := config.Spec.TelemetryStore.Status.Extras["_overrides"]

	configBuf := bytes.NewBuffer(nil)
	if err := ConfigClickhousev2556YAML.Execute(configBuf, data); err != nil {
		return fmt.Errorf("failed to execute config template: %w", err)
	}

	functionBuf := bytes.NewBuffer(nil)
	if err := FunctionsClickhousev2556YAML.Execute(functionBuf, data); err != nil {
		return fmt.Errorf("failed to execute config template: %w", err)
	}

	base := configBuf.String()

	if overrides != "" {
		merged, err := types.MergeYAML(base, overrides)
		if err != nil {
			return fmt.Errorf("failed to merge config overrides for config.yaml: %w", err)
		}
		base = merged
	}

	config.Spec.TelemetryStore.Status.Config.Data = map[string]string{
		"config.yaml":    base,
		"functions.yaml": functionBuf.String(),
	}

	return nil
}

func (molding *telemetrystore) getData(config *v1alpha1.Casting) (Data, error) {
	storeAddresses := config.Spec.TelemetryStore.Status.Addresses.TCP
	if len(storeAddresses) == 0 {
		return Data{}, fmt.Errorf("telemetry store addresses not set in status")
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
		return Data{}, fmt.Errorf(
			"insufficient addresses: have %d, need %d (shards=%d × replicas=%d)",
			len(storeAddresses), expectedNodes, shardCount, replicaCount,
		)
	}

	newStoreAddrs, err := types.NewAddresses(storeAddresses[:expectedNodes])
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse addresses: %w", err)
	}

	keeperAddresses := config.Spec.TelemetryKeeper.Status.Addresses.Client
	if len(keeperAddresses) == 0 {
		return Data{}, fmt.Errorf("telemetry keeper addresses not set in status")
	}

	newKeeperAddrs, err := types.NewAddresses(keeperAddresses)
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse addresses: %w", err)
	}

	return Data{
		StoreAddresses:  newStoreAddrs,
		KeeperAddresses: newKeeperAddrs,
		ShardCount:      shardCount,
		ReplicaCount:    replicaCount,
	}, nil
}
