package foundry

import (
	"context"
	"log/slog"
	"slices"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

// Uncast uncasts one document. The caller walks the set backwards, against the
// order the lock records, so a workload leaves before the substrate it runs on.
func (foundry *Foundry) Uncast(ctx context.Context, machinery v1alpha1.Machinery, poursPath string) error {
	p, err := foundry.Plan(ctx, machinery)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return foundryerrors.Wrapf(ctx.Err(), foundryerrors.TypeInternal, "failed to uncast %s: the run was interrupted", machinery.Name())
	}

	foundry.Logger.InfoContext(ctx, "uncasting",
		slog.String("casting.kind", machinery.Kind().String()),
		slog.String("casting.metadata.name", machinery.Name()))

	return p.Uncast(ctx, poursPath)
}
