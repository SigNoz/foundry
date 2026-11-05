package foundry

import (
	"context"
	"log/slog"
	"slices"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/planner"
)

// Down plans every document of the set, then takes them down in the reverse
// of the cast order. A document that fails stops the run at
// machineries[len(machineries)-1-n]; what already came down stays down.
func (foundry *Foundry) Down(ctx context.Context, machineries []v1alpha1.Machinery, poursPath string) (int, error) {
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

	return n, nil
}
