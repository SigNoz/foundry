package systemdbinarycasting

import (
	"bytes"
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.MoldingEnricher = (*systemdBinaryMoldingEnricher)(nil)

type systemdBinaryMoldingEnricher struct{}

func newSystemdBinaryMoldingEnricher(config *collectionagent.Casting) *systemdBinaryMoldingEnricher {
	if config.Metadata.Annotations == nil {
		config.Metadata.Annotations = map[string]string{}
	}
	config.Metadata.Annotations[collectionagent.CollectorAgentBinaryPath.Key] = collectionagent.CollectorAgentBinaryPath.Resolve(config.Metadata.Annotations)

	return &systemdBinaryMoldingEnricher{}
}

func (e *systemdBinaryMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
	if kind != v1alpha1.MoldingKindCollector {
		return nil
	}

	replicas := 1
	if cluster := config.Spec.Collector.Spec.Cluster; cluster.Replicas != nil {
		replicas = *cluster.Replicas
	}

	if replicas != 1 {
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "failed to enrich the collector: spec.collector.spec.cluster.replicas is %d, a systemd unit runs once per host", replicas)
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
