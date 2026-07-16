package infrastructure

import (
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting/infrastructure/awskubernetesterraformcasting"
	infrastructurecasting "github.com/signoz/foundry/internal/casting/infrastructure/casting"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
)

type CastingItem struct {
	Casting infrastructurecasting.Casting
	Toolers []tooler.Tooler
}

type Registry struct {
	castings map[v1alpha1.TypeDeployment]CastingItem
}

func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		castings: map[v1alpha1.TypeDeployment]CastingItem{
			{
				Platform: v1alpha1.PlatformAWS,
				Mode:     v1alpha1.ModeKubernetes,
				Flavor:   v1alpha1.FlavorTerraform,
			}: {
				Casting: awskubernetesterraformcasting.New(logger),
				Toolers: []tooler.Tooler{terraformtooler.New()},
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

func (registry *Registry) Casting(deployment v1alpha1.TypeDeployment) (infrastructurecasting.Casting, error) {
	item, ok := registry.lookup(deployment)
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "infrastructure deployment '%+v' is not supported", deployment)
	}
	return item.Casting, nil
}

func (registry *Registry) Toolers(deployment v1alpha1.TypeDeployment) ([]tooler.Tooler, error) {
	item, ok := registry.lookup(deployment)
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "infrastructure deployment '%+v' is not supported", deployment)
	}
	return item.Toolers, nil
}
