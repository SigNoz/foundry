package casting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/tooler"
)

// DeploymentDir is the subdirectory within the pours directory where
// Installation Kind materials (compose files, service units, configs) are written.
const DeploymentDir = "deployment"

type Casting interface {
	// Returns the enricher for the casting.
	Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error)

	// Generates all the files needed for casting.
	Forge(ctx context.Context, config installation.Casting, poursPath string) ([]domain.Material, error)

	// Runs the forged files. Toolers are the tool interfaces the registry
	// lists for this casting.
	Cast(ctx context.Context, config installation.Casting, poursPath string, toolers []tooler.Tooler) error

	// Removes what Cast deployed: definitions only, never data, never
	// users, config stays.
	Melt(ctx context.Context, config installation.Casting, poursPath string, toolers []tooler.Tooler) error
}
