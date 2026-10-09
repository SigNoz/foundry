package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/planner"
	"github.com/signoz/foundry/internal/writer"
)

// Forge plans every document of the casting file, forges them, records them
// all in one lock, and writes their materials. A document that fails stops the
// run at machineries[n], so no lock is written for a set that did not forge
// whole.
func (foundry *Foundry) Forge(ctx context.Context, machineries []v1alpha1.Machinery, path string, poursWriterOpts *writer.Options) (int, error) {
	planners := make([]planner.Planner, 0, len(machineries))

	for i, machinery := range machineries {
		p, err := foundry.Plan(ctx, machinery)
		if err != nil {
			return i, err
		}

		planners = append(planners, p)
	}

	materials := []domain.Material{}
	n := 0

	for _, p := range planners {
		machinery := p.Machinery()

		foundry.Logger.InfoContext(ctx, "forging casting",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		for _, kind := range p.MoldingKinds() {
			if err := p.EnrichStatus(ctx, kind); err != nil {
				return n, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to enrich molding %s", kind)
			}
		}

		for _, kind := range p.MoldingKinds() {
			foundry.Logger.InfoContext(ctx, "molding configuration for kind", slog.String("molding.kind", kind.String()))
			if err := p.Mold(ctx, kind); err != nil {
				return n, err
			}
		}

		if err := p.MergeStatusIntoSpec(); err != nil {
			return n, err
		}

		forged, err := p.Forge(ctx, poursWriterOpts.TargetDirectory)
		if err != nil {
			return n, err
		}

		for _, pe := range p.Patches() {
			patcher, ok := foundry.Patchers[pe.PatchType()]
			if !ok {
				return n, foundryerrors.Newf(foundryerrors.TypeUnsupported, "unknown patch type %q", pe.PatchType())
			}
			foundry.Logger.InfoContext(ctx, "applying patch", slog.String("patch.type", pe.PatchType()), slog.String("patch.target", pe.Target))
			forged, err = patcher.Apply(ctx, forged, pe)
			if err != nil {
				return n, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to apply patch for target %q", pe.Target)
			}
		}

		materials = append(materials, forged...)
		n++
	}

	// The moldings resolve each casting in place, so the set that arrived is the
	// set the lock records. It is written once every document has forged.
	foundry.Logger.InfoContext(ctx, "writing lock file")
	if err := foundry.Config.CreateV1Alpha1Lock(ctx, machineries, path); err != nil {
		return n, err
	}

	if len(materials) == 0 {
		foundry.Logger.WarnContext(ctx, "castings did not generate any materials for writing")
		return n, nil
	}

	poursWriter, err := writer.New(foundry.Logger, poursWriterOpts)
	if err != nil {
		return n, err
	}

	foundry.Logger.InfoContext(ctx, "writing materials", slog.Int("count", len(materials)))
	return n, poursWriter.WriteMany(ctx, materials...)
}
