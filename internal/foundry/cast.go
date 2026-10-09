package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/planner"
)

// Cast plans every document of the set, then casts them in the order the lock
// records. A document that fails stops the run at machineries[n]; what already
// cast stays cast.
func (foundry *Foundry) Cast(ctx context.Context, machineries []v1alpha1.Machinery, poursPath string) (int, error) {
	planners := make([]planner.Planner, 0, len(machineries))

	for i, machinery := range machineries {
		p, err := foundry.Plan(ctx, machinery)
		if err != nil {
			return i, err
		}

		planners = append(planners, p)
	}

	n := 0

	for _, p := range planners {
		machinery := p.Machinery()

		foundry.Logger.InfoContext(ctx, "casting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		if err := p.Cast(ctx, poursPath); err != nil {
			return n, err
		}

		n++
	}

	return n, nil
}
