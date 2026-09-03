package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
)

// Gauge proves a tool once across documents: proving one is a statement about
// the machine, not about a document.
func (foundry *Foundry) Gauge(ctx context.Context, machineries []v1alpha1.Machinery) error {
	planners, err := foundry.Plan(ctx, machineries)
	if err != nil {
		return err
	}

	proven := map[string]struct{}{}

	for _, p := range planners {
		machinery := p.Machinery()

		foundry.Logger.InfoContext(ctx, "gauging",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		for _, tool := range p.Toolers() {
			if _, done := proven[tool.Name()]; done {
				continue
			}

			if err := tool.Gauge(ctx); err != nil {
				return err
			}

			proven[tool.Name()] = struct{}{}

			foundry.Logger.InfoContext(ctx, "tool is available", slog.String("tool.name", tool.Name()))
		}
	}

	return nil
}
