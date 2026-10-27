package foundry

import (
	"context"
	"log/slog"
	"slices"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/planner"
)

// Down refuses a casting file that does not match its lock, then plans every
// document and takes them down in the reverse of the lock order, deleting the
// lock once every document came down. A document that fails stops the run at
// machineries[len(machineries)-1-n]; what already came down stays down and the
// lock stays.
func (foundry *Foundry) Down(ctx context.Context, machineries []v1alpha1.Machinery, path string, poursPath string) (int, error) {
	resolvers := make([]planner.Planner, 0, len(machineries))

	for _, machinery := range machineries {
		p, err := foundry.Plan(ctx, machinery)
		if err != nil {
			return len(machineries), err
		}

		resolvers = append(resolvers, p)
	}

	for _, p := range resolvers {
		machinery := p.Machinery()

		foundry.Logger.InfoContext(ctx, "resolving casting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		for _, kind := range p.MoldingKinds() {
			if err := p.EnrichStatus(ctx, kind); err != nil {
				return len(machineries), foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to enrich molding %s", kind)
			}
		}

		for _, kind := range p.MoldingKinds() {
			if err := p.Mold(ctx, kind); err != nil {
				return len(machineries), err
			}
		}

		if err := p.MergeStatusIntoSpec(); err != nil {
			return len(machineries), err
		}
	}

	matched, err := foundry.Config.MatchV1Alpha1Lock(ctx, machineries, path)
	if err != nil {
		return len(machineries), err
	}

	if !matched {
		return len(machineries), foundryerrors.Newf(foundryerrors.TypeInvalidInput, "casting file %q changed since its lock was forged: restore the file to take down what the lock recorded, or forge again to take down what it now states", path)
	}

	planners := make([]planner.Planner, len(machineries))

	for i, machinery := range slices.Backward(machineries) {
		p, err := foundry.Plan(ctx, machinery)
		if err != nil {
			return len(machineries) - 1 - i, err
		}

		planners[i] = p
	}

	n := 0

	for _, p := range slices.Backward(planners) {
		machinery := p.Machinery()

		foundry.Logger.InfoContext(ctx, "taking down",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		if err := p.Down(ctx, poursPath); err != nil {
			return n, err
		}

		n++
	}

	foundry.Logger.InfoContext(ctx, "deleting lock file")

	if err := foundry.Config.DeleteV1Alpha1Lock(ctx, path); err != nil {
		return n, err
	}

	return n, nil
}
