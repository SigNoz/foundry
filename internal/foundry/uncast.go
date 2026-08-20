package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
)

// Uncast uncasts one document. The caller walks the set backwards, against the
// order the lock records, so a workload leaves before the substrate it runs on.
func (foundry *Foundry) Uncast(ctx context.Context, machinery v1alpha1.Machinery, poursPath string) error {
	p, err := foundry.Plan(ctx, machinery)
	if err != nil {
		return err
	}

	foundry.Logger.InfoContext(ctx, "uncasting",
		slog.String("casting.kind", machinery.Kind().String()),
		slog.String("casting.metadata.name", machinery.Name()))

	return p.Uncast(ctx, poursPath)
}
