package signozmolding

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
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
	if config.Spec.Signoz.Status.Env == nil {
		config.Spec.Signoz.Status.Env = make(map[string]string)
	}

	if config.Spec.Signoz.Spec.Env == nil {
		config.Spec.Signoz.Spec.Env = make(map[string]string)
	}

	// Add telemetry store addresses
	config.Spec.Signoz.Status.Env["SIGNOZ_TELEMETRYSTORE_PROVIDER"] = config.Spec.TelemetryStore.Kind.String()

	if val, ok := config.Spec.Signoz.Spec.Env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"]; ok {
		molding.logger.WarnContext(ctx, "SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN is going to be overridden", slog.String("value", val))
	}

	config.Spec.Signoz.Status.Env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"] = strings.Join(config.Spec.TelemetryStore.Status.Addresses.TCP, ",")

	// Add metastore addresses
	config.Spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_PROVIDER"] = config.Spec.MetaStore.Kind.String()

	if val, ok := config.Spec.Signoz.Spec.Env["SIGNOZ_SQLSTORE_POSTGRES_DSN"]; ok {
		molding.logger.WarnContext(ctx, "SIGNOZ_SQLSTORE_POSTGRES_DSN is going to be overridden", slog.String("value", val))
	}

	// construct postgres dsn with user, password, host, port, and db
	addrs, err := types.NewAddresses(config.Spec.MetaStore.Status.Addresses.DSN)
	if err != nil {
		return fmt.Errorf("failed to parse addresses: %w", err)
	}

	var dsns []string
	user := config.Spec.MetaStore.Status.Env["POSTGRES_USER"]
	password := config.Spec.MetaStore.Status.Env["POSTGRES_PASSWORD"]
	db := config.Spec.MetaStore.Status.Env["POSTGRES_DB"]
	for _, addr := range addrs {
		dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, password, addr.Host(), addr.Port(), db)
		dsns = append(dsns, dsn)
	}

	config.Spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_POSTGRES_DSN"] = strings.Join(dsns, ",")

	return nil
}
