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

// EnrichStatus contributes nothing yet: the docker telemetry additions
// (receivers, resource attribution) land separately.
func (e *dockerComposeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	return nil
}
