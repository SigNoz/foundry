package foundry

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

func (f *Foundry) Gauge(ctx context.Context, machinery v1alpha1.Machinery) error {
	p, err := newPlanner(ctx, machinery, f.Logger)
	if err != nil {
		return err
	}
	return p.Gauge(ctx)
}
