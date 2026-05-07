package foundry

import (
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/casting/coolifycasting"
	"github.com/signoz/foundry/internal/casting/dockercomposecasting"
	"github.com/signoz/foundry/internal/casting/dockerswarmcasting"
	"github.com/signoz/foundry/internal/casting/ecsterraformcasting"
	"github.com/signoz/foundry/internal/casting/kuberneteshelmcasting"
	"github.com/signoz/foundry/internal/casting/kuberneteskustomizecasting"
	"github.com/signoz/foundry/internal/casting/railwaytemplatecasting"
	"github.com/signoz/foundry/internal/casting/rendercasting"
	"github.com/signoz/foundry/internal/casting/systemdcasting"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/clickhousekeepertooler"
	"github.com/signoz/foundry/internal/tooler/clickhousetooler"
	"github.com/signoz/foundry/internal/tooler/dockercomposetooler"
	"github.com/signoz/foundry/internal/tooler/dockerswarmtooler"
	"github.com/signoz/foundry/internal/tooler/dockertooler"
	"github.com/signoz/foundry/internal/tooler/helmtooler"
	"github.com/signoz/foundry/internal/tooler/kubectltooler"
	"github.com/signoz/foundry/internal/tooler/postgrestooler"
	"github.com/signoz/foundry/internal/tooler/systemdtooler"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
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
				Mode:   v1alpha1.ModeDocker,
				Flavor: v1alpha1.FlavorCompose,
			}: {
				Casting: dockercomposecasting.New(logger),
				Toolers: []tooler.Tooler{dockertooler.New(), dockercomposetooler.New()},
			},
			{
				Mode:   v1alpha1.ModeSystemd,
				Flavor: v1alpha1.FlavorBinary,
			}: {
				Casting: systemdcasting.New(logger),
				Toolers: []tooler.Tooler{systemdtooler.New(), clickhousekeepertooler.New(), clickhousetooler.New(), postgrestooler.New()},
			},
			{
				Mode:   v1alpha1.ModeDocker,
				Flavor: v1alpha1.FlavorSwarm,
			}: {
				Casting: dockerswarmcasting.New(logger),
				Toolers: []tooler.Tooler{dockertooler.New(), dockerswarmtooler.New()},
			},
			{
				Mode:   v1alpha1.ModeKubernetes,
				Flavor: v1alpha1.FlavorKustomize,
			}: {
				Casting: kuberneteskustomizecasting.New(logger),
				Toolers: []tooler.Tooler{kubectltooler.New()},
			},
			{
				Platform: v1alpha1.PlatformRender,
				Flavor:   v1alpha1.FlavorBlueprint,
			}: {
				Casting: rendercasting.New(logger),
			},
			{
				Platform: v1alpha1.PlatformCoolify,
				Flavor:   v1alpha1.FlavorStack,
			}: {
				Casting: coolifycasting.New(logger),
			},
			{
				Platform: v1alpha1.PlatformRailway,
				Flavor:   v1alpha1.FlavorTemplate,
			}: {
				Casting: railwaytemplatecasting.New(logger),
			},
			{
				Platform: v1alpha1.PlatformECS,
				Flavor:   v1alpha1.FlavorTerraform,
				Mode:     v1alpha1.ModeEC2,
			}: {
				Casting: ecsterraformcasting.New(logger),
				Toolers: []tooler.Tooler{terraformtooler.New()},
			},
			{
				Mode:   v1alpha1.ModeKubernetes,
				Flavor: v1alpha1.FlavorHelm,
			}: {
				Casting: kuberneteshelmcasting.New(logger),
				Toolers: []tooler.Tooler{helmtooler.New()},
			},
		},
	}, nil
}

func (registry *Registry) CastingItems() map[v1alpha1.TypeDeployment]CastingItem {
	return registry.castings
}

func (registry *Registry) lookup(deployment v1alpha1.TypeDeployment) (CastingItem, bool) {
	if item, ok := registry.castings[deployment]; ok {
		return item, true
	}
	// Fall back to matching without platform (platform may be set for infra generation
	// but the casting itself is platform-agnostic, e.g. docker/compose on aws).
	if deployment.Platform != (v1alpha1.Platform{}) {
		item, ok := registry.castings[v1alpha1.TypeDeployment{Mode: deployment.Mode, Flavor: deployment.Flavor}]
		return item, ok
	}
	return CastingItem{}, false
}

func (registry *Registry) Casting(deployment v1alpha1.TypeDeployment) (casting.Casting, error) {
	item, ok := registry.lookup(deployment)
	if !ok {
		return nil, fmt.Errorf("deployment '%+v' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this deployment", deployment)
	}

	return item.Casting, nil
}

func (registry *Registry) Toolers(deployment v1alpha1.TypeDeployment) ([]tooler.Tooler, error) {
	item, ok := registry.lookup(deployment)
	if !ok {
		return nil, fmt.Errorf("deployment '%+v' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this deployment", deployment)
	}

	return item.Toolers, nil
}
