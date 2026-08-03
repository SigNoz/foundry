package dockercomposecasting

import (
	"bytes"
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*dockerComposeMoldingEnricher)(nil)

type dockerComposeMoldingEnricher struct{}

func newDockerComposeMoldingEnricher() *dockerComposeMoldingEnricher {
	return &dockerComposeMoldingEnricher{}
}

func (e *dockerComposeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	if kind != v1alpha1.MoldingKindCollector {
		return nil
	}

	if config.Spec.Collector.Kind != collectionagent.CollectorKindAgent {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	if err := agentYAMLTemplate.Execute(buf, nil); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute agent template")
	}

	config.Spec.Collector.Status.Config.Set(config.Spec.Collector.Kind.ConfigKey(), buf.Bytes())

	return nil
}
