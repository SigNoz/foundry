package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/writer"
)

// Forge runs one casting document through the pipeline (enrich, mold, merge
// status into spec, forge, patch), pours its materials, and records it in the
// lock.
func (foundry *Foundry) Forge(ctx context.Context, machinery v1alpha1.Machinery, path string, poursWriterOpts *writer.Options) error {
	p, err := foundry.Plan(ctx, machinery)
	if err != nil {
		return err
	}

	foundry.Logger.InfoContext(ctx, "forging casting",
		slog.String("casting.kind", machinery.Kind().String()),
		slog.String("casting.metadata.name", machinery.Name()))

	for _, kind := range p.MoldingKinds() {
		if err := p.EnrichStatus(ctx, kind); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to enrich molding %s", kind)
		}
	}

	for _, kind := range p.MoldingKinds() {
		foundry.Logger.InfoContext(ctx, "molding configuration for kind", slog.String("molding.kind", kind.String()))
		if err := p.Mold(ctx, kind); err != nil {
			return err
		}
	}

	if err := p.MergeStatusIntoSpec(); err != nil {
		return err
	}

	forged, err := p.Forge(ctx, poursWriterOpts.TargetDirectory)
	if err != nil {
		return err
	}

	for _, pe := range p.Patches() {
		patcher, ok := foundry.Patchers[pe.PatchType()]
		if !ok {
			return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unknown patch type %q", pe.PatchType())
		}
		foundry.Logger.InfoContext(ctx, "applying patch", slog.String("patch.type", pe.PatchType()), slog.String("patch.target", pe.Target))
		forged, err = patcher.Apply(ctx, forged, pe)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to apply patch for target %q", pe.Target)
		}
	}

	if len(forged) == 0 {
		foundry.Logger.WarnContext(ctx, "casting did not generate any materials for writing")
	} else {
		poursWriter, err := writer.New(foundry.Logger, poursWriterOpts)
		if err != nil {
			return err
		}

		foundry.Logger.InfoContext(ctx, "writing materials", slog.Int("count", len(forged)))
		if err := poursWriter.WriteMany(ctx, forged...); err != nil {
			return err
		}
	}

	return foundry.Config.CreateOrUpdateV1Alpha1Lock(ctx, machinery, path)
}
