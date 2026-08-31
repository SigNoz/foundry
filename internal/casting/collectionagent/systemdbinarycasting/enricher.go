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

// newSystemdBinaryMoldingEnricher records each annotation's resolved value so
// the lock captures the full resolved config: user-set values win, absent
// ones fall back to the default.
func newSystemdBinaryMoldingEnricher(config *collectionagent.Casting) *systemdBinaryMoldingEnricher {
	if config.Metadata.Annotations == nil {
		config.Metadata.Annotations = map[string]string{}
	}
	for _, a := range collectionagent.Annotations() {
		config.Metadata.Annotations[a.Key] = a.Resolve(config.Metadata.Annotations)
	}

	return &systemdBinaryMoldingEnricher{}
}

func (e *systemdBinaryMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *collectionagent.Casting) error {
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
