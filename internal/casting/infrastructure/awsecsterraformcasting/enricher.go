package awsecsterraformcasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
	"github.com/signoz/foundry/internal/molding/infrastructure/resourcemolding"
)

// Each class takes the smallest size of its SigNoz capacity guide family, so a
// substrate boots cheaply in any region; the casting states production sizing.
const (
	machineTypePersistent = "c5.large"
	machineTypeEphemeral  = "t3.medium"
	volumeType            = "gp3"
)

var _ infrastructuremolding.MoldingEnricher = (*awsEcsTerraformMoldingEnricher)(nil)

type awsEcsTerraformMoldingEnricher struct{}

func newAwsEcsTerraformMoldingEnricher() *awsEcsTerraformMoldingEnricher {
	return &awsEcsTerraformMoldingEnricher{}
}

// EnrichStatus omits subnets: an availability zone is per-account.
func (e *awsEcsTerraformMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *infrastructure.Casting) error {
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
