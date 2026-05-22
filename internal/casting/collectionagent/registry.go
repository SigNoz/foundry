package collectionagent

import (
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

type CastingItem struct {
	Casting Casting
	Toolers []tooler.Tooler
}

type Registry struct {
	castings map[v1alpha1.TypeDeployment]CastingItem
}

func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		castings: map[v1alpha1.TypeDeployment]CastingItem{},
	}
}

func (r *Registry) lookup(deployment v1alpha1.TypeDeployment) (CastingItem, bool) {
	if item, ok := r.castings[deployment]; ok {
		return item, true
	}
	if deployment.Platform != (v1alpha1.Platform{}) {
		item, ok := r.castings[v1alpha1.TypeDeployment{Mode: deployment.Mode, Flavor: deployment.Flavor}]
		return item, ok
	}
	return CastingItem{}, false
}

func (r *Registry) Casting(deployment v1alpha1.TypeDeployment) (Casting, error) {
	item, ok := r.lookup(deployment)
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "collectionagent deployment '%+v' is not supported", deployment)
	}
	return item.Casting, nil
}

func (r *Registry) Toolers(deployment v1alpha1.TypeDeployment) ([]tooler.Tooler, error) {
	item, ok := r.lookup(deployment)
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "collectionagent deployment '%+v' is not supported", deployment)
	}
	return item.Toolers, nil
}
