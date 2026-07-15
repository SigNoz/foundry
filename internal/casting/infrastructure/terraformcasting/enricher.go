package terraformcasting

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

var _ infrastructuremolding.MoldingEnricher = (*enricher)(nil)

type enricher struct {
	logger *slog.Logger
}

func (e *enricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *infrastructure.Casting) error {
	return nil
}
