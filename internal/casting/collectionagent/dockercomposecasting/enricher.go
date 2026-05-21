package dockercomposecasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*dockerComposeMoldingEnricher)(nil)

type dockerComposeMoldingEnricher struct{}

func newDockerComposeMoldingEnricher() *dockerComposeMoldingEnricher {
	return &dockerComposeMoldingEnricher{}
}

func (e *dockerComposeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	// CollectionAgent's collector molding has no inter-component addresses to
	// derive today; enrichment is a no-op. When the collector needs to know
	// the SigNoz ingest endpoint (or peer collectors), populate it here.
	return nil
}
