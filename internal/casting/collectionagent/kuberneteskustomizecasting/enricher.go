package kuberneteskustomizecasting

import (
	"bytes"
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*kubernetesKustomizeMoldingEnricher)(nil)

type kubernetesKustomizeMoldingEnricher struct{}

func newKubernetesKustomizeMoldingEnricher() *kubernetesKustomizeMoldingEnricher {
	return &kubernetesKustomizeMoldingEnricher{}
}

func (e *kubernetesKustomizeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	if kind != v1alpha1.MoldingKindCollector {
		return nil
	}

	var tmpl *domain.Template
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindAgent:
		tmpl = agentYAMLTemplate
	case collectionagent.CollectorKindDeployment:
		tmpl = deploymentYAMLTemplate
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
	}

	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, config); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", config.Spec.Collector.Kind)
	}

	config.Spec.Collector.Status.Config.Set(config.Spec.Collector.Kind.ConfigKey(), buf.Bytes())

	return nil
}
