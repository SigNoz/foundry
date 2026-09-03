package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

// Cast casts every document of the set, in the order the lock records. A
// document that fails stops the run; what already cast stays cast.
func (foundry *Foundry) Cast(ctx context.Context, machineries []v1alpha1.Machinery, poursPath string) error {
	planners, err := foundry.Plan(ctx, machineries)
	if err != nil {
		return err
	}

	for _, p := range planners {
		machinery := p.Machinery()

		if ctx.Err() != nil {
			return foundryerrors.Wrapf(ctx.Err(), foundryerrors.TypeInternal, "failed to cast %s: the run was interrupted", machinery.Name())
		}

		foundry.Logger.InfoContext(ctx, "casting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		if err := p.Cast(ctx, poursPath); err != nil {
			return err
		}
	}

	return nil
}
