package foundry

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/writer"
)

func (foundry *Foundry) Forge(ctx context.Context, machinery v1alpha1.Machinery, path string, poursWriterOpts *writer.Options) error {
	switch c := machinery.(type) {
	case *installation.Casting:
		return foundry.forgeInstallation(ctx, *c, path, poursWriterOpts)
	case *collectionagent.Casting:
		return foundry.forgeCollectionAgent(ctx, *c, path, poursWriterOpts)
	}
	return fmt.Errorf("unsupported casting kind %q", machinery.Kind())
}

func (foundry *Foundry) forgeCollectionAgent(ctx context.Context, config collectionagent.Casting, path string, _ *writer.Options) error {
	foundry.Logger.WarnContext(ctx, "collectionagent forge not yet implemented; writing lock only",
		slog.String("casting.metadata.name", config.Metadata.Name))
	return foundry.Config.CreateV1Alpha1Lock(ctx, &config, path)
}

func (foundry *Foundry) forgeInstallation(ctx context.Context, config installation.Casting, path string, poursWriterOpts *writer.Options) error {
	foundry.Logger.InfoContext(ctx, "starting forge pipeline", slog.String("casting.metadata.name", config.Metadata.Name))

	spec := &config.Spec

	casting, err := foundry.Registry.Casting(spec.Deployment)
	if err != nil {
		foundry.Logger.ErrorContext(ctx, "casting not found", slog.String("casting.spec.deployment.mode", spec.Deployment.Mode.String()))
		return err
	}

	foundry.Logger.InfoContext(ctx, "enriching moldings with casting specific information", slog.String("casting.metadata.name", config.Metadata.Name))
	moldingEnricher, err := casting.Enricher(ctx, &config)
	if err != nil {
		foundry.Logger.ErrorContext(ctx, "failed to get molding enricher", slog.String("casting.metadata.name", config.Metadata.Name), foundryerrors.LogAttr(err))
		return fmt.Errorf("failed to get molding enricher: %w", err)
	}

	foundry.Logger.InfoContext(ctx, "enriching configuration with casting specific information", slog.String("casting.metadata.name", config.Metadata.Name))
	for _, moldingKind := range molding.MoldingsInOrder() {
		err = moldingEnricher.EnrichStatus(ctx, moldingKind, &config)
		if err != nil {
			return fmt.Errorf("failed to enrich configuration with casting specific information: %w", err)
		}
	}

	// Molding the configuration
	for _, molding := range molding.MoldingsInOrder() {
		foundry.Logger.InfoContext(ctx, "molding configuration for kind", slog.String("molding.kind", molding.String()))
		err = foundry.Moldings[molding].MoldV1Alpha1(ctx, &config)
		if err != nil {
			foundry.Logger.ErrorContext(ctx, "failed to mold configuration", slog.String("molding.kind", molding.String()), foundryerrors.LogAttr(err))
			return err
		}
	}
	// merging status into spec
	foundry.Logger.InfoContext(ctx, "merging status into spec", slog.String("casting.metadata.name", config.Metadata.Name))
	if err := config.MergeStatusIntoSpec(); err != nil {
		foundry.Logger.ErrorContext(ctx, "failed to merge status into spec", slog.String("casting.metadata.name", config.Metadata.Name), foundryerrors.LogAttr(err))
		return err
	}

	// Forging the configuration
	foundry.Logger.InfoContext(ctx, "forging configuration with the merged spec and generating materials", slog.String("casting.metadata.name", config.Metadata.Name))
	materials, err := casting.Forge(ctx, config, poursWriterOpts.TargetDirectory)
	if err != nil {
		return err
	}

	// Apply patch operations from spec.patches
	for _, pe := range spec.Patches {
		patcher, ok := foundry.Patchers[pe.PatchType()]
		if !ok {
			return fmt.Errorf("unknown patch type %q", pe.PatchType())
		}
		foundry.Logger.InfoContext(ctx, "applying patch", slog.String("casting.metadata.name", config.Metadata.Name), slog.String("patch.type", pe.PatchType()), slog.String("patch.target", pe.Target))
		materials, err = patcher.Apply(ctx, materials, pe)
		if err != nil {
			foundry.Logger.ErrorContext(ctx, "failed to apply patch", slog.String("casting.metadata.name", config.Metadata.Name), slog.String("patch.target", pe.Target), foundryerrors.LogAttr(err))
			return fmt.Errorf("failed to apply patch for target %q: %w", pe.Target, err)
		}
	}

	// Generate infrastructure-as-code manifests if enabled, before writing the lock file
	// so that the generated file contents are captured in the lock's infrastructure.status.
	var infraMaterials []domain.Material
	if spec.Infrastructure.Enabled {
		foundry.Logger.InfoContext(ctx, "generating infrastructure manifests",
			slog.String("casting.metadata.name", config.Metadata.Name),
			slog.String("deployment.platform", spec.Deployment.Platform.String()))

		infraMaterials, err = foundry.InfrastructureGenerator.Generate(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to generate infrastructure manifests: %w", err)
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
	foundry.Logger.InfoContext(ctx, "writing lock file", slog.String("casting.metadata.name", config.Metadata.Name))
	if err = foundry.Config.CreateV1Alpha1Lock(ctx, &config, path); err != nil {
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
		if err = poursWriter.WriteMany(ctx, infraMaterials...); err != nil {
			return err
		}
	}

	// Writing the deployment materials
	foundry.Logger.InfoContext(ctx, "writing materials", slog.String("casting.metadata.name", config.Metadata.Name))
	if err = poursWriter.WriteMany(ctx, materials...); err != nil {
		return err
	}

	return nil
}
