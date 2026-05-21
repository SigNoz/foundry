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
)

// planner is the per-Kind contract Foundry iterates against. Every Kind
// expresses itself in the same vocabulary:
//
//   - identity:   Machinery, Patches
//   - ordering:   MoldingKinds (the moldings this Kind processes, in order)
//   - stages:     EnrichStatus, Mold, MergeStatusIntoSpec, Forge
//   - lifecycle:  Cast, Gauge
type planner interface {
	Machinery() v1alpha1.Machinery
	Patches() []v1alpha1.PatchEntry

	MoldingKinds() []v1alpha1.MoldingKind
	EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind) error
	Mold(ctx context.Context, kind v1alpha1.MoldingKind) error
	MergeStatusIntoSpec() error

	Forge(ctx context.Context, target string) ([]domain.Material, error)
	Cast(ctx context.Context, poursPath string) error
	Gauge(ctx context.Context) error
}

var (
	_ planner = (*installationcasting.Planner)(nil)
	_ planner = (*collectionagentcasting.Planner)(nil)
)

// newPlanner is the single dispatch site. Adding a new Kind means adding one
// case here and one concrete Planner in that Kind's package.
func newPlanner(ctx context.Context, m v1alpha1.Machinery, logger *slog.Logger) (planner, error) {
	switch c := m.(type) {
	case *installation.Casting:
		return installationcasting.NewPlanner(ctx, c, logger)
	case *collectionagent.Casting:
		return collectionagentcasting.NewPlanner(ctx, c, logger)
	}
	return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported casting kind %q", m.Kind())
}
