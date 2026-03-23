package infrastructure

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

// Generator is the interface for infrastructure-as-code generators.
// Implementations produce IaC manifests (e.g., Terraform, Pulumi) from a casting configuration
// and can validate the generated output using the underlying tool.
type Generator interface {
	// Generate produces IaC materials from the casting configuration.
	Generate(ctx context.Context, config v1alpha1.Casting) ([]types.Material, error)

	// Validate runs the IaC tool's built-in validation (e.g., terraform validate)
	// against the manifests written to poursPath.
	Validate(ctx context.Context, poursPath string) error
}
