package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
)

// Cast casts one casting document into the target environment.
func (foundry *Foundry) Cast(ctx context.Context, machinery v1alpha1.Machinery, poursPath string) error {
	p, err := foundry.Plan(ctx, machinery)
	if err != nil {
		return err
	}

	foundry.Logger.InfoContext(ctx, "casting",
		slog.String("casting.kind", machinery.Kind().String()),
		slog.String("casting.metadata.name", machinery.Name()))

	return p.Cast(ctx, poursPath)
}
