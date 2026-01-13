package telemetrykeepermolding

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
	defaultServerCount = 1
)

var _ molding.Molding = (*telemetrykeeper)(nil)

type telemetrykeeper struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *telemetrykeeper {
	return &telemetrykeeper{
		logger: logger,
	}
}

func (molding *telemetrykeeper) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindTelemetryKeeper
}

func (molding *telemetrykeeper) MoldV1Alpha1(ctx context.Context, config *v1alpha1.Casting) error {
	data, err := molding.getData(config)
	if err != nil {
		molding.logger.ErrorContext(ctx, "failed to get data", foundryerrors.LogAttr(err))
		return err
	}

	configs := make(map[string]string, len(data.TelemetryKeeperClickhouseCluster.Servers))
	for _, server := range data.TelemetryKeeperClickhouseCluster.Servers {
		configBuf := bytes.NewBuffer(nil)
		data.ServerID = server.ID
		if err := KeeperClickhousev2556YAML.Execute(configBuf, data); err != nil {
			return fmt.Errorf("failed to execute keeper template for server %d: %w", server.ID, err)
		}
		configs[fmt.Sprintf("keeper-%d.yaml", server.ID)] = configBuf.String()
	}

	config.Spec.TelemetryKeeper.Spec.Config.Data = configs
	return nil
}

func (molding *telemetrykeeper) getData(config *v1alpha1.Casting) (Data, error) {
	addresses := config.Spec.TelemetryKeeper.Status.Addresses
	if len(addresses) == 0 {
		return Data{}, fmt.Errorf("keeper addresses not set in status")
	}

	raftConfig, err := molding.getTelemetryKeeperCluster(config.Spec.TelemetryKeeper.Spec.Cluster, addresses)
	if err != nil {
		return Data{}, fmt.Errorf("failed to build raft config: %w", err)
	}

	return Data{
		TelemetryKeeperClickhouseCluster: raftConfig,
	}, nil
}

func (molding *telemetrykeeper) getTelemetryKeeperCluster(cluster v1alpha1.TypeCluster, addresses []string) (RaftConfig, error) {
	serverCount := defaultServerCount
	if cluster.Replicas != nil {
		serverCount = *cluster.Replicas
	}

	if len(addresses) < serverCount {
		return RaftConfig{}, fmt.Errorf(
			"insufficient addresses: have %d, need %d servers",
			len(addresses), serverCount,
		)
	}

	parsedAddrs, err := types.ParseAddresses(addresses[:serverCount])
	if err != nil {
		return RaftConfig{}, fmt.Errorf("failed to parse addresses: %w", err)
	}

	servers := make([]Server, 0, serverCount)
	for i, addr := range parsedAddrs {
		servers = append(servers, Server{
			ID:   i + 1,
			Host: addr.Host,
			Port: addr.Port,
		})
	}

	return RaftConfig{Servers: servers}, nil
}
