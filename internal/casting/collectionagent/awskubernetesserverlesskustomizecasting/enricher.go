package awskubernetesserverlesskustomizecasting

import (
	"bytes"
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*awsKubernetesServerlessKustomizeMoldingEnricher)(nil)

type awsKubernetesServerlessKustomizeMoldingEnricher struct{}

func newAwsKubernetesServerlessKustomizeMoldingEnricher() *awsKubernetesServerlessKustomizeMoldingEnricher {
	return &awsKubernetesServerlessKustomizeMoldingEnricher{}
}

func (e *awsKubernetesServerlessKustomizeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	if kind != v1alpha1.MoldingKindCollector {
		return nil
	}

	if !config.Spec.Collector.Spec.IsEnabled() {
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "spec.collector.spec.enabled is false, the CollectionAgent kind requires an enabled collector")
	}

	replicas := 1
	if cluster := config.Spec.Collector.Spec.Cluster; cluster.Replicas != nil {
		replicas = *cluster.Replicas
	}

	var tmpl *domain.Template
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindDeployment:
		if replicas != 1 {
			return foundryerrors.Newf(foundryerrors.TypeUnsupported, "spec.collector.spec.cluster.replicas is %d, one collector scrapes every node", replicas)
		}

		tmpl = deploymentYAMLTemplate
	case collectionagent.CollectorKindSidecar:
		if replicas != 1 {
			return foundryerrors.Newf(foundryerrors.TypeUnsupported, "spec.collector.spec.cluster.replicas is %d, a sidecar runs once in every pod it joins", replicas)
		}

		tmpl = sidecarYAMLTemplate
	default:
		return nil
	}

	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, templateDataFor(*config)); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", config.Spec.Collector.Kind)
	}

	config.Spec.Collector.Status.Config.Set(config.Spec.Collector.Kind.ConfigKey(), buf.Bytes())

	return nil
}
