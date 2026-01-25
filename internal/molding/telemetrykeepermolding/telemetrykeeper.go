package telemetrykeepermolding

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
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
	data, err := newData(config)
	if err != nil {
		molding.logger.ErrorContext(ctx, "failed to get data", foundryerrors.LogAttr(err))
		return err
	}

	// Generate per-server configs (each keeper node needs its own server_id)
	configs := make(map[string]string, data.ServerCount)
	for i := 0; i < data.ServerCount; i++ {
		configBuf := bytes.NewBuffer(nil)
		data.ServerID = i // 0-indexed, used for array indexing in template
		if err := KeeperClickhousev2556YAML.Execute(configBuf, data); err != nil {
			return fmt.Errorf("failed to execute keeper template for server %d: %w", data.ServerID, err)
		}
		configs[fmt.Sprintf("keeper-%d.yaml", i)] = configBuf.String()
	}

	config.Spec.TelemetryKeeper.Status.Config.Data = configs
	return nil
}
