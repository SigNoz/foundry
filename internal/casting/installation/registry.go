package installation

import (
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
	foundryerrors "github.com/signoz/foundry/internal/errors"
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

type CastingItem struct {
	Casting casting.Casting
	Toolers []tooler.Tooler
}

type Registry struct {
	castings map[v1alpha1.TypeDeployment]CastingItem
}

func NewRegistry(logger *slog.Logger) *Registry {
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
	}
}

func (r *Registry) CastingItems() map[v1alpha1.TypeDeployment]CastingItem {
	return r.castings
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

func (r *Registry) Casting(deployment v1alpha1.TypeDeployment) (casting.Casting, error) {
	item, ok := r.lookup(deployment)
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "deployment '%+v' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this deployment", deployment)
	}
	return item.Casting, nil
}

func (r *Registry) Toolers(deployment v1alpha1.TypeDeployment) ([]tooler.Tooler, error) {
	item, ok := r.lookup(deployment)
	if !ok {
		return nil, foundryerrors.Newf(foundryerrors.TypeUnsupported, "deployment '%+v' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this deployment", deployment)
	}
	return item.Toolers, nil
}
