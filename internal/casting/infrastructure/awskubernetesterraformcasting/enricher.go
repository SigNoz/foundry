package awskubernetesterraformcasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

var _ infrastructuremolding.MoldingEnricher = (*awsKubernetesTerraformMoldingEnricher)(nil)

type awsKubernetesTerraformMoldingEnricher struct{}

func newAwsKubernetesTerraformMoldingEnricher() *awsKubernetesTerraformMoldingEnricher {
	return &awsKubernetesTerraformMoldingEnricher{}
}

func (e *awsKubernetesTerraformMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *infrastructure.Casting) error {
	return nil
}
