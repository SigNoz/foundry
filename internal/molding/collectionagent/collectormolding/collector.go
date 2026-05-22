package collectormolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.Molding = (*collector)(nil)

type collector struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *collector {
	return &collector{logger: logger}
}

func (m *collector) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindCollector
}

// MoldV1Alpha1 dispatches on Spec.Collector.Kind. Per-kind logic lives in
// the kind's own file (agent.go for CollectorKindAgent).
func (m *collector) MoldV1Alpha1(ctx context.Context, config *collectionagent.Casting) error {
	switch config.Spec.Collector.Kind {
	case collectionagent.CollectorKindAgent:
		return m.moldAgent(config)
	}
	return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", config.Spec.Collector.Kind)
}
