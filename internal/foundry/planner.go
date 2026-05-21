package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	collectionagentcasting "github.com/signoz/foundry/internal/casting/collectionagent"
	installationcasting "github.com/signoz/foundry/internal/casting/installation"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

// planner is the per-Kind contract Foundry iterates against. Every Kind
// expresses itself in the same vocabulary:
//
//   - identity:   Machinery, Patches, Toolers
//   - ordering:   MoldingKinds (the moldings this Kind processes, in order)
//   - stages:     EnrichStatus, Mold, MergeStatusIntoSpec
//   - lifecycle:  Forge, Cast
type planner interface {
	Machinery() v1alpha1.Machinery
	Patches() []v1alpha1.PatchEntry
	Toolers() []tooler.Tooler

	MoldingKinds() []v1alpha1.MoldingKind
	EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind) error
	Mold(ctx context.Context, kind v1alpha1.MoldingKind) error
	MergeStatusIntoSpec() error

	Forge(ctx context.Context, target string) ([]domain.Material, error)
	Cast(ctx context.Context, poursPath string) error
}

var (
	_ planner = (*installationcasting.Planner)(nil)
	_ planner = (*collectionagentcasting.Planner)(nil)
)

// plannerFactories maps each Kind to its planner constructor. Adding a new
// Kind means adding one entry here.
var plannerFactories = map[v1alpha1.Kind]func(context.Context, v1alpha1.Machinery, *slog.Logger) (planner, error){
	v1alpha1.KindInstallation: func(ctx context.Context, m v1alpha1.Machinery, logger *slog.Logger) (planner, error) {
		return installationcasting.NewPlanner(ctx, m.(*installation.Casting), logger)
	},
	v1alpha1.KindCollectionAgent: func(ctx context.Context, m v1alpha1.Machinery, logger *slog.Logger) (planner, error) {
		return collectionagentcasting.NewPlanner(ctx, m.(*collectionagent.Casting), logger)
	},
}

func newPlanner(ctx context.Context, m v1alpha1.Machinery, logger *slog.Logger) (planner, error) {
	factory, ok := plannerFactories[m.Kind()]
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported casting kind %q", m.Kind())
	}
	return factory(ctx, m, logger)
}
