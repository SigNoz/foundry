package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
)

// Uncast uncasts every document of the set, against the order the lock
// records: a workload leaves before the substrate it runs on. A document that
// fails stops the run; what already went stays gone.
func (foundry *Foundry) Uncast(ctx context.Context, machineries []v1alpha1.Machinery, poursPath string) error {
	planners, err := foundry.Plan(ctx, machineries)
	if err != nil {
		return err
	}

	for i := len(planners) - 1; i >= 0; i-- {
		p := planners[i]
		machinery := p.Machinery()

		foundry.Logger.InfoContext(ctx, "uncasting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		if err := p.Uncast(ctx, poursPath); err != nil {
			return err
		}
	}

	return nil
}
