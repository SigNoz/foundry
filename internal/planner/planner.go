package planner

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/tooler"
)

// Planner is the per-Kind contract Foundry iterates against. Every Kind
// expresses itself in the same vocabulary:
//
//   - identity:   Machinery, Patches, Toolers
//   - ordering:   MoldingKinds (the moldings this Kind processes, in order)
//   - stages:     EnrichStatus, Mold, MergeStatusIntoSpec
//   - lifecycle:  Forge, Cast
type Planner interface {
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
