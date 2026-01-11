package telemetrystoremolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
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

func (molding *telemetrystore) MoldV1Alpha1(ctx context.Context, config *v1alpha1.Casting) error {
	if !config.Spec.TelemetryStore.Spec.Enabled {
		return nil
	}

	telemetrystoreSpec := DefaultSpec()
	if err := v1alpha1.Merge(telemetrystoreSpec, config.Spec.TelemetryStore.Spec); err != nil {
		return err
	}

	// Set the merged telemetry store spec
	config.Spec.TelemetryStore.Spec = telemetrystoreSpec

	// Add keeper addresses

	return nil
}
