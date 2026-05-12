package signozmolding

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
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
	spec := config.SigNozSpec()

	if spec.Signoz.Status.Env == nil {
		spec.Signoz.Status.Env = make(map[string]string)
	}

	if spec.Signoz.Spec.Env == nil {
		spec.Signoz.Spec.Env = make(map[string]string)
	}

	// Add telemetry store addresses
	spec.Signoz.Status.Env["SIGNOZ_TELEMETRYSTORE_PROVIDER"] = spec.TelemetryStore.Kind.String()

	if val, ok := spec.Signoz.Spec.Env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"]; ok {
		molding.logger.WarnContext(ctx, "SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN is going to be overridden", slog.String("value", val))
	}

	spec.Signoz.Status.Env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"] = strings.Join(spec.TelemetryStore.Status.Addresses.TCP, ",")

	// Add metastore addresses
	spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_PROVIDER"] = spec.MetaStore.Kind.String()

	switch spec.MetaStore.Kind {
	case v1alpha1.MetaStoreKindSQLite:
		spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_SQLITE_PATH"] = "/var/lib/signoz/signoz.db"
	case v1alpha1.MetaStoreKindPostgres:
		if spec.MetaStore.Status.Addresses.DSN != nil {
			if val, ok := spec.Signoz.Spec.Env["SIGNOZ_SQLSTORE_POSTGRES_DSN"]; ok {
				molding.logger.WarnContext(ctx, "SIGNOZ_SQLSTORE_POSTGRES_DSN is going to be overridden", slog.String("value", val))
			}
			// construct postgres dsn with user, password, host, port, and db
			addrs, err := domain.ParseAddresses(spec.MetaStore.Status.Addresses.DSN)
			if err != nil {
				return fmt.Errorf("failed to parse addresses: %w", err)
			}
			var dsns []string
			user := spec.MetaStore.Status.Env["POSTGRES_USER"]
			password := spec.MetaStore.Status.Env["POSTGRES_PASSWORD"]
			db := spec.MetaStore.Status.Env["POSTGRES_DB"]
			for _, addr := range addrs {
				dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, password, addr.Host(), addr.Port(), db)
				dsns = append(dsns, dsn)
			}
			spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_POSTGRES_DSN"] = strings.Join(dsns, ",")
		}
	}
	return nil
}
