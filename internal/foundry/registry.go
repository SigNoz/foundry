package foundry

import (
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/casting/dockercomposecasting"
	"github.com/signoz/foundry/internal/casting/systemdcasting"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/clickhousekeepertooler"
	"github.com/signoz/foundry/internal/tooler/clickhousetooler"
	"github.com/signoz/foundry/internal/tooler/dockercomposetooler"
	"github.com/signoz/foundry/internal/tooler/dockertooler"
	"github.com/signoz/foundry/internal/tooler/postgrestooler"
	"github.com/signoz/foundry/internal/tooler/systemdtooler"
)

// Defines a single casting item in the registry.
type CastingItem struct {
	// The particular casting implementation.
	Casting casting.Casting

	// The toolers for the particular casting.
	Toolers []tooler.Tooler
}

type Registry struct {
	// Castings for the different deployments.
	castings map[v1alpha1.TypeDeployment]CastingItem
}

func NewRegistry(logger *slog.Logger) (*Registry, error) {
	return &Registry{
		castings: map[v1alpha1.TypeDeployment]CastingItem{
			{
				Mode:   "docker",
				Flavor: "compose",
			}: {
				Casting: dockercomposecasting.New(logger),
				Toolers: []tooler.Tooler{dockertooler.New(), dockercomposetooler.New()},
			},
			{
				Mode:   "systemd",
				Flavor: "binary",
			}: {
				Casting: systemdcasting.New(logger),
				Toolers: []tooler.Tooler{systemdtooler.New(), clickhousekeepertooler.New(), clickhousetooler.New(), postgrestooler.New()},
			},
		},
	}, nil
}

func (registry *Registry) CastingItems() map[v1alpha1.TypeDeployment]CastingItem {
	return registry.castings
}

func (registry *Registry) Casting(deployment v1alpha1.TypeDeployment) (casting.Casting, error) {
	item, ok := registry.castings[deployment]
	if !ok {
		return nil, fmt.Errorf("deployment '%+v' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this deployment", deployment)
	}

	return item.Casting, nil
}

func (registry *Registry) Toolers(deployment v1alpha1.TypeDeployment) ([]tooler.Tooler, error) {
	item, ok := registry.castings[deployment]
	if !ok {
		return nil, fmt.Errorf("deployment '%+v' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this deployment", deployment)
	}

	return item.Toolers, nil
}
