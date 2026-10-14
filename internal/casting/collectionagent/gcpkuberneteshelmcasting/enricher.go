package gcpkuberneteshelmcasting

import (
	"bytes"
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*gcpKubernetesHelmMoldingEnricher)(nil)

type gcpKubernetesHelmMoldingEnricher struct {
	generic collectionagentmolding.MoldingEnricher
}

func newGcpKubernetesHelmMoldingEnricher(generic collectionagentmolding.MoldingEnricher) *gcpKubernetesHelmMoldingEnricher {
	return &gcpKubernetesHelmMoldingEnricher{generic: generic}
}

func (e *gcpKubernetesHelmMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	if err := e.generic.EnrichStatus(ctx, kind, config); err != nil {
		return err
	}

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
	if err := tmpl.Execute(buf, nil); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", config.Spec.Collector.Kind)
	}

	// Lists replace on merge, so the delta states the whole detector list.
	key := config.Spec.Collector.Kind.ConfigKey()

	merged, err := domain.MergeYAML(config.Spec.Collector.Status.Config.Data[key], buf.String())
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to merge %s resource detection", config.Spec.Collector.Kind)
	}

	config.Spec.Collector.Status.Config.Set(key, []byte(merged))

	return nil
}
