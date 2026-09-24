package awseksfargatekustomizecasting

import (
	"bytes"
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*awsEksFargateKustomizeMoldingEnricher)(nil)

type awsEksFargateKustomizeMoldingEnricher struct{}

func newAwsEksFargateKustomizeMoldingEnricher() *awsEksFargateKustomizeMoldingEnricher {
	return &awsEksFargateKustomizeMoldingEnricher{}
}

func (e *awsEksFargateKustomizeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	if kind != v1alpha1.MoldingKindCollector {
		return nil
	}

	replicas := 1
	if cluster := config.Spec.Collector.Spec.Cluster; cluster.Replicas != nil {
		replicas = *cluster.Replicas
	}

	if replicas != 1 {
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "failed to enrich the collector: spec.collector.spec.cluster.replicas is %d, one collector scrapes every node", replicas)
	}

	if config.Spec.Collector.Kind != collectionagent.CollectorKindDeployment {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	if err := deploymentYAMLTemplate.Execute(buf, templateDataFor(*config)); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", config.Spec.Collector.Kind)
	}

	config.Spec.Collector.Status.Config.Set(config.Spec.Collector.Kind.ConfigKey(), buf.Bytes())

	return nil
}
