package metastoremolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/errors"
)

type metastore struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *metastore {
	return &metastore{
		logger: logger,
	}
}

func (molding *metastore) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindMetaStore
}

func (molding *metastore) MoldV1Alpha1(ctx context.Context, config *installation.Casting) error {
	if config.Spec.MetaStore.Status.Env == nil {
		config.Spec.MetaStore.Status.Env = make(map[string]string)
	}

	if config.Spec.MetaStore.Spec.Env == nil {
		config.Spec.MetaStore.Spec.Env = make(map[string]string)
	}

	switch config.Spec.MetaStore.Kind {
	case installation.MetaStoreKindSQLite:
		replicas := config.Spec.MetaStore.Spec.Cluster.Replicas
		if replicas != nil && *replicas != 1 {
			return errors.Newf(errors.TypeInvalidInput, "metastore.spec.cluster.replicas must be 1 when metastore.kind is sqlite; sqlite is embedded and per-instance storage is driven by signoz.spec.cluster.replicas (got %d)", *replicas)
		}
	case installation.MetaStoreKindPostgres:
		if val, ok := config.Spec.MetaStore.Spec.Env["POSTGRES_DB"]; ok {
			molding.logger.WarnContext(ctx, "POSTGRES_DB is going to be overridden", slog.String("value", val))
		}

		config.Spec.MetaStore.Status.Env["POSTGRES_DB"] = "signoz"

		if val, ok := config.Spec.MetaStore.Spec.Env["POSTGRES_USER"]; ok {
			molding.logger.WarnContext(ctx, "POSTGRES_USER is going to be overridden", slog.String("value", val))
		}

		config.Spec.MetaStore.Status.Env["POSTGRES_USER"] = "signoz"

		if val, ok := config.Spec.MetaStore.Spec.Env["POSTGRES_PASSWORD"]; ok {
			molding.logger.WarnContext(ctx, "POSTGRES_PASSSWORD is going to be overridden", slog.String("value", val))
		}

		config.Spec.MetaStore.Status.Env["POSTGRES_PASSWORD"] = "signoz"
	}

	return nil
}
