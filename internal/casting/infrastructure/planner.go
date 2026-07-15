package infrastructure

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/planner"
	"github.com/signoz/foundry/internal/tooler"
)

var _ planner.Planner = (*Planner)(nil)

// Planner is the Infrastructure Kind's per-Kind orchestrator. The Kind has no
// moldings: reference validation happens at planner construction and all
// derivation happens inside the casting at forge.
type Planner struct {
	config  *infrastructure.Casting
	logger  *slog.Logger
	casting Casting
	toolers []tooler.Tooler
}

func NewPlanner(ctx context.Context, c *infrastructure.Casting, logger *slog.Logger) (planner.Planner, error) {
	registry := NewRegistry(logger)

	castingStrategy, err := registry.Casting(c.Spec.Deployment)
	if err != nil {
		return nil, err
	}

	toolers, err := registry.Toolers(c.Spec.Deployment)
	if err != nil {
		return nil, err
	}

	return &Planner{
		config:  c,
		logger:  logger,
		casting: castingStrategy,
		toolers: toolers,
	}, nil
}

func (p *Planner) Machinery() v1alpha1.Machinery  { return p.config }
func (p *Planner) Patches() []v1alpha1.PatchEntry { return p.config.Spec.Patches }

func (p *Planner) MoldingKinds() []v1alpha1.MoldingKind { return nil }

func (p *Planner) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind) error {
	return foundryerrors.Newf(foundryerrors.TypeInternal, "infrastructure has no moldings")
}

func (p *Planner) Mold(ctx context.Context, kind v1alpha1.MoldingKind) error {
	return foundryerrors.Newf(foundryerrors.TypeInternal, "infrastructure has no moldings")
}

func (p *Planner) MergeStatusIntoSpec() error {
	return p.config.MergeStatusIntoSpec()
}

func (p *Planner) Forge(ctx context.Context, target string) ([]domain.Material, error) {
	return p.casting.Forge(ctx, *p.config, target)
}

func (p *Planner) Cast(ctx context.Context, poursPath string) error {
	return p.casting.Cast(ctx, *p.config, poursPath)
}

func (p *Planner) Toolers() []tooler.Tooler { return p.toolers }
