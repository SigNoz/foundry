package collectormolding

import (
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"

	"context"
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

func (m *collector) MoldV1Alpha1(ctx context.Context, config *collectionagent.Casting) error {
	if config.Spec.Collector.Status.Env == nil {
		config.Spec.Collector.Status.Env = make(map[string]string)
	}
	config.Spec.Collector.Status.Env["OTEL_COLLECTOR_KIND"] = config.Spec.Collector.Kind.String()
	return nil
}
