package foundry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/writer"
)

func (foundry *Foundry) Forge(ctx context.Context, config v1alpha1.Casting, path string, poursWriterOpts *writer.Options) error {
	foundry.Logger.InfoContext(ctx, "starting forging pipeline", slog.String("casting.metadata.name", config.Metadata.Name))

	// Get the casting for the deployment mode
	casting, err := foundry.CastingByDeploymentMode(config.Spec.Deployment.Mode)
	if err != nil {
		foundry.Logger.ErrorContext(ctx, "casting not found:", slog.String("casting.spec.deployment.mode", config.Spec.Deployment.Mode))
		return fmt.Errorf("casting not found: %w", err)
	}

	// Enrich the configuration with casting specific information
	foundry.Logger.InfoContext(ctx, "getting the molding enricher", slog.String("casting.metadata.name", config.Metadata.Name))
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
	if err := v1alpha1.MergeCastingSpecAndStatus(&config); err != nil {
		foundry.Logger.ErrorContext(ctx, "failed to merge status into spec", slog.String("casting.metadata.name", config.Metadata.Name), foundryerrors.LogAttr(err))
		return err
	}

	// Forging the configuration
	foundry.Logger.InfoContext(ctx, "forging configuration with the merged spec and generating materials", slog.String("casting.metadata.name", config.Metadata.Name))
	materials, err := casting.Forge(ctx, config, poursWriterOpts.TargetDirectory)
	if err != nil {
		return err
	}

	// writing the merged config to the config file
	foundry.Logger.InfoContext(ctx, "writing lock file", slog.String("casting.metadata.name", config.Metadata.Name))

	err = foundry.Config.CreateV1Alpha1Lock(ctx, config, path)
	if err != nil {
		return err
	}

	if len(materials) == 0 {
		foundry.Logger.WarnContext(ctx, "casting did not generate any materials for writing")
		return nil
	}

	poursWriter, err := writer.New(foundry.Logger, poursWriterOpts)
	if err != nil {
		return err
	}

	// Writing the materials
	foundry.Logger.InfoContext(ctx, "writing materials", slog.String("casting.metadata.name", config.Metadata.Name))
	err = poursWriter.WriteMany(ctx, materials...)
	if err != nil {
		return err
	}

	return nil
}
