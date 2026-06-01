package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/mechanic"
)

// Inspect resolves a mechanic resource path against the loaded casting and runs the targeted inspection.
func (foundry *Foundry) Inspect(ctx context.Context, machinery v1alpha1.Machinery, resource mechanic.Resource, overrides mechanic.Overrides) error {
	resolved, err := resource.Resolve(machinery.Kind())
	if err != nil {
		return err
	}

	if resolved.EntityKind == "" || resolved.EntityID == "" {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "inspect requires <molding> <entity-kind> <entity-id>, got %q", resolved.String())
	}

	foundry.Logger.InfoContext(
		ctx, "mechanic inspect target resolved",
		slog.String("kind", resolved.Kind.String()),
		slog.String("molding", resolved.Molding.String()),
		slog.String("entity.kind", resolved.EntityKind),
		slog.String("entity.id", resolved.EntityID),
		slog.Bool("override.signoz", overrides.Signoz != ""),
		slog.Bool("override.clickhouse", overrides.ClickhouseDSN != ""),
		slog.Bool("override.metastore", overrides.MetastoreDSN != ""),
	)

	return nil
}
