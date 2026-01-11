package signozmolding

import (
	"context"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.Molding = (*signoz)(nil)

type signoz struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *signoz {
	return &signoz{
		logger: logger,
	}
}

func (molding *signoz) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindSignoz
}

func (molding *signoz) MoldV1Alpha1(ctx context.Context, config *v1alpha1.Casting) error {
	signoz := Default()
	if err := v1alpha1.Merge(signoz, config.Spec.Signoz); err != nil {
		return err
	}

	// Set the merged signoz spec
	config.Spec.Signoz = *signoz

	if !config.Spec.Signoz.Spec.Enabled {
		return nil
	}

	// Add telemetry store addresses
	if config.Spec.TelemetryStore.Spec.Enabled {
		config.Spec.Signoz.Status.Env["SIGNOZ_TELEMETRYSTORE_PROVIDER"] = config.Spec.TelemetryStore.Kind.String()

		if val, ok := config.Spec.Signoz.Spec.Env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"]; ok {
			molding.logger.WarnContext(ctx, "SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN is going to be overridden", slog.String("value", val))
		}

		config.Spec.Signoz.Status.Env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"] = strings.Join(config.Spec.TelemetryStore.Status.Addresses, ",")
	}

	// Add metastore addresses
	if config.Spec.MetaStore.Spec.Enabled {
		config.Spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_PROVIDER"] = config.Spec.MetaStore.Kind.String()

		if val, ok := config.Spec.Signoz.Spec.Env["SIGNOZ_SQLSTORE_POSTGRES_DSN"]; ok {
			molding.logger.WarnContext(ctx, "SIGNOZ_SQLSTORE_POSTGRES_DSN is going to be overridden", slog.String("value", val))
		}

		config.Spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_POSTGRES_DSN"] = strings.Join(config.Spec.MetaStore.Status.Addresses, ",")
	}

	return nil
}
