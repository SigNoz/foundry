package kuberneteshelmcasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.MoldingEnricher = (*helmMoldingEnricher)(nil)

type helmMoldingEnricher struct{}

func newHelmMoldingEnricher(config *v1alpha1.Casting) *helmMoldingEnricher {
	return &helmMoldingEnricher{}
}

func (e *helmMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	return nil
}
