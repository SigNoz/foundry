package molding

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
)

// MoldingEnricher populates a molding's Status fields from the surrounding
// installation casting. The ordering of EnrichStatus calls is owned by the
// installation Planner, which iterates the kinds it knows about.
type MoldingEnricher interface {
	EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error
}

// Molding generates materials for a single SigNoz component. Mutates the
// config in place; not safe for concurrent use.
type Molding interface {
	Kind() v1alpha1.MoldingKind
	MoldV1Alpha1(ctx context.Context, config *installation.Casting) error
}
