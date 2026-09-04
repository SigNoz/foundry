package infrastructure

import (
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting/infrastructure/ecsec2terraformcasting"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
)

type CastingItem struct {
	Casting Casting
	Toolers []tooler.Tooler
}

type Registry struct {
	// Castings for the different deployments.
	castings map[v1alpha1.TypeDeployment]CastingItem
}

func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		castings: map[v1alpha1.TypeDeployment]CastingItem{
			{
				Platform: v1alpha1.PlatformECS,
				Mode:     v1alpha1.ModeEC2,
				Flavor:   v1alpha1.FlavorTerraform,
			}: {
				Casting: ecsec2terraformcasting.New(logger),
				Toolers: []tooler.Tooler{terraformtooler.New(logger)},
			},
		},
	}
}

// lookup matches the exact deployment; each platform, mode, and flavor
// combination registers its own casting.
func (registry *Registry) lookup(deployment v1alpha1.TypeDeployment) (CastingItem, bool) {
	item, ok := registry.castings[deployment]
	return item, ok
}

func (registry *Registry) Casting(deployment v1alpha1.TypeDeployment) (Casting, error) {
	item, ok := registry.lookup(deployment)
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "infrastructure deployment '%+v' is not supported", deployment)
	}
	return item.Casting, nil
}

func (registry *Registry) Toolers(deployment v1alpha1.TypeDeployment) ([]tooler.Tooler, error) {
	item, ok := registry.lookup(deployment)
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "infrastructure deployment '%+v' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this deployment", deployment)
	}
	return item.Toolers, nil
}
