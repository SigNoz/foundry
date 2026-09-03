package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

func (foundry *Foundry) Cast(ctx context.Context, machinery v1alpha1.Machinery, poursPath string) error {
	p, err := foundry.Plan(ctx, machinery)
	if err != nil {
		return err
	}

	foundry.Logger.InfoContext(ctx, "casting",
		slog.String("casting.kind", machinery.Kind().String()),
		slog.String("casting.metadata.name", machinery.Name()))

	if ctx.Err() != nil {
		return foundryerrors.Wrapf(ctx.Err(), foundryerrors.TypeInternal, "failed to cast %s: the run was interrupted", machinery.Name())
	}

	return p.Cast(ctx, poursPath)
}
