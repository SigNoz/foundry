package metastoremolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
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

func (molding *metastore) MoldV1Alpha1(ctx context.Context, config *v1alpha1.Casting) error {
	return nil
}
