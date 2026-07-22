package foundry

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

func (foundry *Foundry) Cast(ctx context.Context, machinery v1alpha1.Machinery, poursPath string) error {
	p, err := foundry.newPlanner(ctx, machinery)
	if err != nil {
		return err
	}
	return p.Cast(ctx, poursPath)
}
