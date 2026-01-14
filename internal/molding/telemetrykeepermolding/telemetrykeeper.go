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

	// Generate per-server configs (each keeper node needs its own server_id)
	configs := make(map[string]string, data.ServerCount)
	for i := 0; i < data.ServerCount; i++ {
		configBuf := bytes.NewBuffer(nil)
		data.ServerID = i
		if err := KeeperClickhousev2556YAML.Execute(configBuf, data); err != nil {
			return fmt.Errorf("failed to execute keeper template for server %d: %w", data.ServerID, err)
		}
		configs[fmt.Sprintf("keeper-%d.yaml", data.ServerID)] = configBuf.String()
	}

	config.Spec.TelemetryKeeper.Spec.Config.Data = configs
	return nil
}

func (molding *telemetrykeeper) getData(config *v1alpha1.Casting) (Data, error) {
	addresses := config.Spec.TelemetryKeeper.Status.Addresses
	if len(addresses) == 0 {
		return Data{}, fmt.Errorf("keeper addresses not set in status")
	}

	cluster := config.Spec.TelemetryKeeper.Spec.Cluster
	serverCount := defaultServerCount
	if cluster.Replicas != nil {
		serverCount = *cluster.Replicas
	}

	if len(addresses) < serverCount {
		return Data{}, fmt.Errorf(
			"insufficient addresses: have %d, need %d servers",
			len(addresses), serverCount,
		)
	}

	newAddrs, err := types.NewAddresses(addresses[:serverCount])
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse addresses: %w", err)
	}

	return Data{
		Addresses:   newAddrs,
		ServerCount: serverCount,
	}, nil
}
