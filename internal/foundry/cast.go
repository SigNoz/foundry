package foundry

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

func (f *Foundry) Cast(ctx context.Context, machinery v1alpha1.Machinery, poursPath string) error {
	p, err := newPlanner(ctx, machinery, f.Logger)
	if err != nil {
		return err
	}
	return p.Cast(ctx, poursPath)
}
