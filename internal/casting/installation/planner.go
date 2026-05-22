package installation

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/molding/ingestermolding"
	"github.com/signoz/foundry/internal/molding/metastoremolding"
	"github.com/signoz/foundry/internal/molding/signozmolding"
	"github.com/signoz/foundry/internal/molding/telemetrykeepermolding"
	"github.com/signoz/foundry/internal/molding/telemetrystoremolding"
	"github.com/signoz/foundry/internal/planner"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
)

var _ planner.Planner = (*Planner)(nil)

// Planner is the Installation Kind's per-Kind orchestrator. It satisfies the
// foundry planner contract by exposing this Kind's moldings, enricher, and
// casting strategy as verbs on a single value.
type Planner struct {
	config   *installation.Casting
	logger   *slog.Logger
	casting  casting.Casting
	toolers  []tooler.Tooler
	enricher molding.MoldingEnricher
	moldings []molding.Molding
}

func NewPlanner(ctx context.Context, c *installation.Casting, logger *slog.Logger) (planner.Planner, error) {
	registry := NewRegistry(logger)

	castingStrategy, err := registry.Casting(c.Spec.Deployment)
	if err != nil {
		return nil, err
	}

	toolers, err := registry.Toolers(c.Spec.Deployment)
	if err != nil {
		return nil, err
	}

	enricher, err := castingStrategy.Enricher(ctx, c)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to get molding enricher")
	}

	moldings := []molding.Molding{
		telemetrykeepermolding.New(logger),
		telemetrystoremolding.New(logger),
		metastoremolding.New(logger),
		signozmolding.New(logger),
		ingestermolding.New(logger),
	}

	return &Planner{
		config:   c,
		logger:   logger,
		casting:  castingStrategy,
		toolers:  toolers,
		enricher: enricher,
		moldings: moldings,
	}, nil
}

func (p *Planner) Machinery() v1alpha1.Machinery  { return p.config }
func (p *Planner) Patches() []v1alpha1.PatchEntry { return p.config.Spec.Patches }

func (p *Planner) MoldingKinds() []v1alpha1.MoldingKind {
	kinds := make([]v1alpha1.MoldingKind, len(p.moldings))
	for i, m := range p.moldings {
		kinds[i] = m.Kind()
	}
	return kinds
}

func (p *Planner) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind) error {
	return p.enricher.EnrichStatus(ctx, kind, p.config)
}

func (p *Planner) Mold(ctx context.Context, kind v1alpha1.MoldingKind) error {
	for _, m := range p.moldings {
		if m.Kind() == kind {
			return m.MoldV1Alpha1(ctx, p.config)
		}
	}
	return foundryerrors.Newf(foundryerrors.TypeInternal, "molding %q not registered for installation planner", kind)
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

func (p *Planner) Toolers() []tooler.Tooler {
	toolers := p.toolers
	if p.config.Spec.Infrastructure.Enabled {
		toolers = append(toolers, terraformtooler.New())
	}
	return toolers
}
