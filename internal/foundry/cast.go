package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
)

// Cast casts every document of the set, in the order the lock records. A
// document that fails stops the run and is returned with its error; what
// already cast stays cast.
func (foundry *Foundry) Cast(ctx context.Context, machineries []v1alpha1.Machinery, poursPath string) (v1alpha1.Machinery, error) {
	for _, machinery := range machineries {
		p, err := foundry.Plan(ctx, machinery)
		if err != nil {
			return machinery, err
		}

		foundry.Logger.InfoContext(ctx, "casting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		if err := p.Cast(ctx, poursPath); err != nil {
			return machinery, err
		}
	}

	return nil, nil
}
