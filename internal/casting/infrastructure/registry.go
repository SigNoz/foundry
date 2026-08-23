package infrastructure

import (
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

type CastingItem struct {
	Casting Casting
}

type Registry struct {
	castings map[v1alpha1.TypeDeployment]CastingItem
}

func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		castings: map[v1alpha1.TypeDeployment]CastingItem{},
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
