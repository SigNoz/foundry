package linuxcasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*linuxMoldingEnricher)(nil)

type linuxMoldingEnricher struct {
	material types.Material
}

func newLinuxMoldingEnricher(config *v1alpha1.Casting) (*linuxMoldingEnricher, error) {

	// Get Services Material
	material, err := getServiceMaterial(config, "signoz.service")
	if err != nil {
		return nil, fmt.Errorf("failed to get signoz service material: %w", err)
	}

	return &linuxMoldingEnricher{material: material}, nil
}

func (enricher *linuxMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	
	// Enrich Status
	return nil
}
