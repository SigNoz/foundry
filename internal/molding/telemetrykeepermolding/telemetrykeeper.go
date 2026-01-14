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

	// Generate per-server configs (each keeper node needs its own server_id)
	configs := make(map[string]string, data.ServerCount)
	for i := 0; i < data.ServerCount; i++ {
		configBuf := bytes.NewBuffer(nil)
		data.ServerID = i
		// data.TcpPort = tcpPorts[i]
		if err := KeeperClickhousev2556YAML.Execute(configBuf, data); err != nil {
			return fmt.Errorf("failed to execute keeper template for server %d: %w", data.ServerID, err)
		}
		configs[fmt.Sprintf("keeper-%d.yaml", data.ServerID)] = configBuf.String()
	}

	config.Spec.TelemetryKeeper.Spec.Config.Data = configs
	return nil
}

func (molding *telemetrykeeper) getData(config *v1alpha1.Casting) (Data, error) {
	// Get server count from cluster spec
	serverCount := max(*config.Spec.TelemetryKeeper.Spec.Cluster.Replicas, 1)

	// Extract addresses from status
	raftAddresses := config.Spec.TelemetryKeeper.Status.Addresses[v1alpha1.TelemetryKeeperRaftAddresses]
	clientAddresses := config.Spec.TelemetryKeeper.Status.Addresses[v1alpha1.TelemetryKeeperClientAddresses]

	// Validate sufficient addresses for server count
	if len(raftAddresses) < serverCount {
		return Data{}, fmt.Errorf(
			"insufficient raft addresses: have %d, need %d servers",
			len(raftAddresses), serverCount,
		)
	}
	if len(clientAddresses) < serverCount {
		return Data{}, fmt.Errorf(
			"insufficient client addresses: have %d, need %d servers",
			len(clientAddresses), serverCount,
		)
	}

	// Parse and validate addresses
	newRaftAddrs, err := types.NewAddresses(raftAddresses[:serverCount])
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse raft addresses: %w", err)
	}

	newClientAddrs, err := types.NewAddresses(clientAddresses[:serverCount])
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse client addresses: %w", err)
	}

	return Data{
		RaftAddresses:   newRaftAddrs,
		ClientAddresses: newClientAddrs,
		ServerCount:     serverCount,
	}, nil
}
