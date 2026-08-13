package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/internal/planner"
)

// Cast casts every document of the set, in the order the lock records. A
// document that fails stops the run; what already cast stays cast.
func (foundry *Foundry) Cast(ctx context.Context, planners []planner.Planner, poursPath string) error {
	for _, p := range planners {
		machinery := p.Machinery()

		foundry.Logger.InfoContext(ctx, "casting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		if err := p.Cast(ctx, poursPath); err != nil {
			return err
		}
	}

	return nil
}
