package foundry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/ux"
	"github.com/signoz/foundry/internal/writer"
)

func (foundry *Foundry) Forge(ctx context.Context, config v1alpha1.Casting, path string, poursWriterOpts *writer.Options) error {
	foundry.UX.Header(fmt.Sprintf("Forging %s (%s/%s)", config.Metadata.Name, config.Spec.Deployment.Mode, config.Spec.Deployment.Flavor))

	casting, err := foundry.Registry.Casting(config.Spec.Deployment)
	if err != nil {
		foundry.Logger.ErrorContext(ctx, "casting not found", slog.String("casting.spec.deployment.mode", config.Spec.Deployment.Mode))
		return err
	}

	// Enrich moldings
	foundry.UX.StartStep("Enriching moldings")
	moldingEnricher, err := casting.Enricher(ctx, &config)
	if err != nil {
		foundry.UX.FinishStep("Enriching moldings", err)
		foundry.Logger.ErrorContext(ctx, "failed to get molding enricher", slog.String("casting.metadata.name", config.Metadata.Name), foundryerrors.LogAttr(err))
		return fmt.Errorf("failed to get molding enricher: %w", err)
	}

	for _, moldingKind := range molding.MoldingsInOrder() {
		err = moldingEnricher.EnrichStatus(ctx, moldingKind, &config)
		if err != nil {
			foundry.UX.FinishStep("Enriching moldings", err)
			return fmt.Errorf("failed to enrich configuration with casting specific information: %w", err)
		}
	}
	foundry.UX.FinishStep("Enriched moldings", nil)

	// Mold configuration
	for _, m := range molding.MoldingsInOrder() {
		foundry.UX.StartStep(fmt.Sprintf("Molding %s", m))
		err = foundry.Moldings[m].MoldV1Alpha1(ctx, &config)
		if err != nil {
			foundry.UX.FinishStep(fmt.Sprintf("Molding %s", m), err)
			foundry.Logger.ErrorContext(ctx, "failed to mold configuration", slog.String("molding.kind", m.String()), foundryerrors.LogAttr(err))
			return err
		}
		foundry.UX.FinishStep(fmt.Sprintf("Molded %s", m), nil)
	}

	// Merge status into spec
	foundry.UX.StartStep("Merging spec and status")
	if err := v1alpha1.MergeCastingSpecAndStatus(&config); err != nil {
		foundry.UX.FinishStep("Merging spec and status", err)
		foundry.Logger.ErrorContext(ctx, "failed to merge status into spec", slog.String("casting.metadata.name", config.Metadata.Name), foundryerrors.LogAttr(err))
		return err
	}
	foundry.UX.FinishStep("Merged spec and status", nil)

	// Generate materials
	foundry.UX.StartStep("Generating materials")
	materials, err := casting.Forge(ctx, config, poursWriterOpts.TargetDirectory)
	if err != nil {
		foundry.UX.FinishStep("Generating materials", err)
		return err
	}
	foundry.UX.FinishStep(fmt.Sprintf("Generated %d materials", len(materials)), nil)

	// Apply patches
	if len(config.Spec.Patches) > 0 {
		foundry.UX.StartStep(fmt.Sprintf("Applying %d patches", len(config.Spec.Patches)))
		for _, pe := range config.Spec.Patches {
			patcher, ok := foundry.Patchers[pe.PatchType()]
			if !ok {
				err := fmt.Errorf("unknown patch type %q", pe.PatchType())
				foundry.UX.FinishStep("Applying patches", err)
				return err
			}
			foundry.Logger.DebugContext(ctx, "applying patch", slog.String("patch.type", pe.PatchType()), slog.String("patch.target", pe.Target))
			materials, err = patcher.Apply(ctx, materials, pe)
			if err != nil {
				foundry.UX.FinishStep("Applying patches", err)
				foundry.Logger.ErrorContext(ctx, "failed to apply patch", slog.String("patch.target", pe.Target), foundryerrors.LogAttr(err))
				return fmt.Errorf("failed to apply patch for target %q: %w", pe.Target, err)
			}
		}
		foundry.UX.FinishStep(fmt.Sprintf("Applied %d patches", len(config.Spec.Patches)), nil)
	}

	// Write lock file
	foundry.UX.StartStep("Writing lock file")
	err = foundry.Config.CreateV1Alpha1Lock(ctx, config, path)
	if err != nil {
		foundry.UX.FinishStep("Writing lock file", err)
		return err
	}
	foundry.UX.FinishStep("Wrote lock file", nil)

	if len(materials) == 0 {
		foundry.Logger.WarnContext(ctx, "casting did not generate any materials for writing")
		return nil
	}

	// Write materials
	foundry.UX.StartStep(fmt.Sprintf("Writing %d files", len(materials)))
	poursWriter, err := writer.New(foundry.Logger, poursWriterOpts)
	if err != nil {
		foundry.UX.FinishStep("Writing files", err)
		return err
	}

	err = poursWriter.WriteMany(ctx, materials...)
	if err != nil {
		foundry.UX.FinishStep("Writing files", err)
		return err
	}
	foundry.UX.FinishStep(fmt.Sprintf("Wrote %d files to %s", len(materials), poursWriterOpts.TargetDirectory), nil)

	// Print file summary
	written := poursWriter.Written()
	uxFiles := make([]ux.WrittenFile, len(written))
	for i, f := range written {
		uxFiles[i] = ux.WrittenFile{Path: f.Path, Size: f.Size}
	}
	foundry.UX.PrintFileSummary(uxFiles)

	return nil
}
