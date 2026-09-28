package ecsfargateterraformcasting

import (
	"bytes"
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*ecsFargateMoldingEnricher)(nil)

type ecsFargateMoldingEnricher struct{}

func newEcsFargateMoldingEnricher() *ecsFargateMoldingEnricher {
	return &ecsFargateMoldingEnricher{}
}

func (e *ecsFargateMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	if kind != v1alpha1.MoldingKindCollector {
		return nil
	}

	replicas := 1
	if cluster := config.Spec.Collector.Spec.Cluster; cluster.Replicas != nil {
		replicas = *cluster.Replicas
	}

	if replicas != 1 {
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "spec.collector.spec.cluster.replicas is %d, a sidecar runs once in every task it joins", replicas)
	}

	if config.Spec.Collector.Kind != collectionagent.CollectorKindSidecar {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	if err := sidecarYAMLTemplate.Execute(buf, nil); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute sidecar template")
	}

	config.Spec.Collector.Status.Config.Set(config.Spec.Collector.Kind.ConfigKey(), buf.Bytes())

	return nil
}
