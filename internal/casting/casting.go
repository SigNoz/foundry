package casting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
)

// DeploymentDir is the subdirectory within the pours directory where
// deployment-specific materials (compose files, service units, configs) are written.
const DeploymentDir = "deployment"

type Casting interface {
	// Returns the enricher for the casting.
	Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error)

	// Generates all the files needed for casting.
	Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error)

	// Runs the forged files.
	Cast(ctx context.Context, config installation.Casting, poursPath string) error
}
