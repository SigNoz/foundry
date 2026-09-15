package ecsterraformcasting

import (
	"bytes"
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*ecsMoldingEnricher)(nil)

type ecsMoldingEnricher struct{}

func newEcsMoldingEnricher() *ecsMoldingEnricher {
	return &ecsMoldingEnricher{}
}

func (e *ecsMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	if kind != v1alpha1.MoldingKindCollector {
		return nil
	}

	if config.Spec.Collector.Kind != collectionagent.CollectorKindAgent {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	if err := agentYAMLTemplate.Execute(buf, agentTemplateDataFor(*config)); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute agent template")
	}

	config.Spec.Collector.Status.Config.Set(config.Spec.Collector.Kind.ConfigKey(), buf.Bytes())

	return nil
}

// Family is the agent's own ECS task family, which its filelog pipeline drops.
type agentTemplateData struct {
	Family string
}

func agentTemplateDataFor(config collectionagent.Casting) agentTemplateData {
	return agentTemplateData{Family: config.Metadata.Name + "-collector-" + config.Spec.Collector.Kind.String()}
}
