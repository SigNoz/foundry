package casting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

// DeploymentDir is the subdirectory within the pours directory where
// deployment-specific materials (compose files, service units, configs) are written.
const DeploymentDir = "deployment"

type Casting interface {
	// Returns the enricher for the casting.
	Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error)

	// Generates all the files needed for casting.
	Forge(ctx context.Context, config v1alpha1.Casting, poursPath string) ([]types.Material, error)

	// Runs the forged files.
	Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error
}
