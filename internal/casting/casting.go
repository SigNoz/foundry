package casting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

type Casting interface {
	// Returns the enricher for the casting.
	Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error)

	// Generates all the files needed for casting.
	Forge(ctx context.Context, config v1alpha1.Casting) ([]types.Material, error)

	// Runs the forged files.
	Cast(ctx context.Context, config v1alpha1.Casting, outputPath string) error
}
