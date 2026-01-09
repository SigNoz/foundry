package casting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

type Casting interface {
	// Generates all the files needed for forging
	Forge(ctx context.Context, config v1alpha1.Casting) (map[string][]byte, error)
}
