package foundry

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/writer"
)

func (foundry *Foundry) Forge(ctx context.Context, machinery v1alpha1.Machinery, path string, poursWriterOpts *writer.Options) error {
	p, err := foundry.newPlanner(ctx, machinery)
	if err != nil {
		return err
	}

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

	materials, err := p.Forge(ctx, poursWriterOpts.TargetDirectory)
	if err != nil {
		return err
	}

	for _, pe := range p.Patches() {
		patcher, ok := foundry.Patchers[pe.PatchType()]
		if !ok {
			return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unknown patch type %q", pe.PatchType())
		}
		foundry.Logger.InfoContext(ctx, "applying patch", slog.String("patch.type", pe.PatchType()), slog.String("patch.target", pe.Target))
		materials, err = patcher.Apply(ctx, materials, pe)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to apply patch for target %q", pe.Target)
		}
	}

	// Generate infrastructure-as-code manifests if enabled, before writing the lock file
	// so that the generated file contents are captured in the lock's infrastructure.status.
	// Gated to installation.Casting
	var infraMaterials []domain.Material
	if config, ok := machinery.(*installation.Casting); ok && config.Spec.Infrastructure.Enabled {
		spec := &config.Spec
		foundry.Logger.InfoContext(ctx, "generating infrastructure manifests",
			slog.String("casting.metadata.name", config.Metadata.Name),
			slog.String("deployment.platform", spec.Deployment.Platform.String()))

		infraMaterials, err = foundry.InfrastructureGenerator.Generate(ctx, *config)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to generate infrastructure manifests")
		}

		// Populate infrastructure status with generated file contents keyed by filename.
		if len(infraMaterials) > 0 {
			spec.Infrastructure.Status = make(map[string]string, len(infraMaterials))
			for _, m := range infraMaterials {
				spec.Infrastructure.Status[filepath.Base(m.Path())] = string(m.FmtContents())
			}
		}
	}

	// writing the merged config (including infrastructure status) to the lock file
	foundry.Logger.InfoContext(ctx, "writing lock file")
	if err := foundry.Config.CreateV1Alpha1Lock(ctx, p.Machinery(), path); err != nil {
		return err
	}

	if len(materials) == 0 && len(infraMaterials) == 0 {
		foundry.Logger.WarnContext(ctx, "casting did not generate any materials for writing")
		return nil
	}

	poursWriter, err := writer.New(foundry.Logger, poursWriterOpts)
	if err != nil {
		return err
	}

	if len(infraMaterials) > 0 {
		foundry.Logger.InfoContext(ctx, "writing infrastructure materials", slog.Int("count", len(infraMaterials)))
		if err := poursWriter.WriteMany(ctx, infraMaterials...); err != nil {
			return err
		}
	}

	foundry.Logger.InfoContext(ctx, "writing materials")
	if err := poursWriter.WriteMany(ctx, materials...); err != nil {
		return err
	}
	return nil
}
