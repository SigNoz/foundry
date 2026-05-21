package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/writer"
)

func (f *Foundry) Forge(ctx context.Context, machinery v1alpha1.Machinery, path string, opts *writer.Options) error {
	p, err := newPlanner(ctx, machinery, f.Logger)
	if err != nil {
		return err
	}

	for _, kind := range p.MoldingKinds() {
		if err := p.EnrichStatus(ctx, kind); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to enrich molding %s", kind)
		}
	}

	for _, kind := range p.MoldingKinds() {
		f.Logger.InfoContext(ctx, "molding configuration for kind", slog.String("molding.kind", kind.String()))
		if err := p.Mold(ctx, kind); err != nil {
			return err
		}
	}

	if err := p.MergeStatusIntoSpec(); err != nil {
		return err
	}

	materials, err := p.Forge(ctx, opts.TargetDirectory)
	if err != nil {
		return err
	}

	for _, pe := range p.Patches() {
		patcher, ok := f.Patchers[pe.PatchType()]
		if !ok {
			return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unknown patch type %q", pe.PatchType())
		}
		f.Logger.InfoContext(ctx, "applying patch", slog.String("patch.type", pe.PatchType()), slog.String("patch.target", pe.Target))
		materials, err = patcher.Apply(ctx, materials, pe)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to apply patch for target %q", pe.Target)
		}
	}

	if len(materials) == 0 {
		f.Logger.WarnContext(ctx, "casting did not generate any materials for writing")
	} else {
		poursWriter, err := writer.New(f.Logger, opts)
		if err != nil {
			return err
		}
		if err := poursWriter.WriteMany(ctx, materials...); err != nil {
			return err
		}
	}

	return f.Config.CreateV1Alpha1Lock(ctx, p.Machinery(), path)
}
