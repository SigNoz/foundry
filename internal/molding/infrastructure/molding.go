package infrastructure

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
)

type MoldingEnricher interface {
	EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *infrastructure.Casting) error
}

type Molding interface {
	Kind() v1alpha1.MoldingKind
	MoldV1Alpha1(ctx context.Context, config *infrastructure.Casting) error
}
