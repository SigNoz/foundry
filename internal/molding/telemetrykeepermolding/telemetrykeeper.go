package telemetrykeepermolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
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
	telemetrykeeper := Default()
	if err := v1alpha1.Merge(telemetrykeeper, config.Spec.TelemetryKeeper); err != nil {
		return err
	}

	// Set the merged telemetry keeper spec
	config.Spec.TelemetryKeeper = *telemetrykeeper

	return nil
}
