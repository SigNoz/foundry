package resourcemolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

var _ infrastructuremolding.Molding = (*resourceMolding)(nil)

type resourceMolding struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *resourceMolding {
	return &resourceMolding{logger: logger}
}

func (molding *resourceMolding) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindResource
}

// MoldV1Alpha1 derives the resource's infrastructure record from the resolved
// reference. Scaffolding: reference resolution and derivation land with the
// forge implementation.
func (molding *resourceMolding) MoldV1Alpha1(ctx context.Context, config *infrastructure.Casting) error {
	return nil
}
