package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
)

// Cast casts every document of the set, in the order the lock records, and
// returns the documents that cast. A document that fails stops the run and is
// the one after them; what already cast stays cast.
func (foundry *Foundry) Cast(ctx context.Context, machineries []v1alpha1.Machinery, poursPath string) ([]v1alpha1.Machinery, error) {
	passed := make([]v1alpha1.Machinery, 0, len(machineries))
	for _, machinery := range machineries {
		p, err := foundry.Plan(ctx, machinery)
		if err != nil {
			return passed, err
		}

		foundry.Logger.InfoContext(ctx, "casting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		if err := p.Cast(ctx, poursPath); err != nil {
			return passed, err
		}

		passed = append(passed, machinery)
	}

	return passed, nil
}
