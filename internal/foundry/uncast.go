package foundry

import (
	"context"
	"log/slog"
	"slices"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

// Uncast uncasts every document of the set, against the order the lock
// records: a workload leaves before the substrate it runs on. A document that
// fails stops the run; what already went stays gone.
func (foundry *Foundry) Uncast(ctx context.Context, machineries []v1alpha1.Machinery, poursPath string) error {
	planners, err := foundry.Plan(ctx, machineries)
	if err != nil {
		return err
	}

	for _, p := range slices.Backward(planners) {
		machinery := p.Machinery()

		if ctx.Err() != nil {
			return foundryerrors.Wrapf(ctx.Err(), foundryerrors.TypeInternal, "failed to uncast %s: the run was interrupted", machinery.Name())
		}

		foundry.Logger.InfoContext(ctx, "uncasting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		if err := p.Uncast(ctx, poursPath); err != nil {
			return err
		}
	}

	return nil
}
