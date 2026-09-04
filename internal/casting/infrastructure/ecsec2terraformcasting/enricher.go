package ecsec2terraformcasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
	"github.com/signoz/foundry/internal/molding/infrastructure/resourcemolding"
)

// Neither machine type is burstable: a store that throttles under sustained
// ingest reads as an outage.
const (
	machineTypePersistent = "m5.large"
	machineTypeEphemeral  = "c5.large"
	volumeType            = "gp3"
)

var _ infrastructuremolding.MoldingEnricher = (*ecsEc2TerraformMoldingEnricher)(nil)

type ecsEc2TerraformMoldingEnricher struct{}

func newEcsEc2TerraformMoldingEnricher() *ecsEc2TerraformMoldingEnricher {
	return &ecsEc2TerraformMoldingEnricher{}
}

// EnrichStatus omits subnets: an availability zone is per-account.
func (e *ecsEc2TerraformMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *infrastructure.Casting) error {
	if kind != v1alpha1.MoldingKindResource {
		return nil
	}

	groups := map[string]infrastructure.ResourceConfigInstanceGroup{
		resourcemolding.GroupPersistent: {
			MachineType: machineTypePersistent,
			RootVolume:  infrastructure.ResourceConfigVolume{Type: volumeType},
			DataVolume:  &infrastructure.ResourceConfigVolume{Type: volumeType},
		},
		resourcemolding.GroupEphemeral: {
			MachineType: machineTypeEphemeral,
			RootVolume:  infrastructure.ResourceConfigVolume{Type: volumeType},
		},
	}

	contribution, err := domain.MarshalYAML(&infrastructure.ResourceConfig{InstanceGroups: groups})
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to marshal resource config contribution")
	}

	config.Spec.Resource.Status.Config.Set(resourcemolding.ResourceConfigName, contribution)

	return nil
}
