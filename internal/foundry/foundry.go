package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	collectionagentcasting "github.com/signoz/foundry/internal/casting/collectionagent"
	installationcasting "github.com/signoz/foundry/internal/casting/installation"
	"github.com/signoz/foundry/internal/config"
	"github.com/signoz/foundry/internal/config/yamlconfig"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/infrastructure"
	terraformgenerator "github.com/signoz/foundry/internal/infrastructure/terraform"
	"github.com/signoz/foundry/internal/patch"
	"github.com/signoz/foundry/internal/patch/jsonpatch"
	"github.com/signoz/foundry/internal/planner"
)

type plannerCtor func(ctx context.Context, m v1alpha1.Machinery, logger *slog.Logger) (planner.Planner, error)

type Foundry struct {
	// Config for loading the casting configuration.
	Config config.Config

	// Patchers for applying patches to generated materials, keyed by patch type.
	Patchers map[string]patch.Patch

	// Logger for logging.
	Logger *slog.Logger

	// Planners for the different casting kinds.
	Planners map[v1alpha1.Kind]plannerCtor

	// InfrastructureGenerator for generating infrastructure-as-code manifests.
	InfrastructureGenerator infrastructure.Generator
}

func New(logger *slog.Logger) (*Foundry, error) {
	return &Foundry{
		Config: yamlconfig.New(),
		Patchers: map[string]patch.Patch{
			v1alpha1.PatchTypeJSONPatch: jsonpatch.New(),
		},
		Logger: logger,
		Planners: map[v1alpha1.Kind]plannerCtor{
			v1alpha1.KindInstallation: func(ctx context.Context, m v1alpha1.Machinery, logger *slog.Logger) (planner.Planner, error) {
				return installationcasting.NewPlanner(ctx, m.(*installation.Casting), logger)
			},
			v1alpha1.KindCollectionAgent: func(ctx context.Context, m v1alpha1.Machinery, logger *slog.Logger) (planner.Planner, error) {
				return collectionagentcasting.NewPlanner(ctx, m.(*collectionagent.Casting), logger)
			},
		},
		InfrastructureGenerator: terraformgenerator.New(logger),
	}, nil
}

func (foundry *Foundry) newPlanner(ctx context.Context, m v1alpha1.Machinery) (planner.Planner, error) {
	ctor, ok := foundry.Planners[m.Kind()]
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported casting kind %q", m.Kind())
	}
	return ctor(ctx, m, foundry.Logger)
}
