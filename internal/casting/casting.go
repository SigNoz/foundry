package casting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/writer"
)

type Casting interface {
	// Generates all the files needed for casting.
	Forge(ctx context.Context, config v1alpha1.Casting) ([]writer.Material, error)

	// Runs the forged files.
	Cast(ctx context.Context, config v1alpha1.Casting) error
}
