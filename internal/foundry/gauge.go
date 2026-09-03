package foundry

import (
	"context"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

// Gauge proves a tool once across documents: proving one is a statement about
// the machine, not about a document. Every tool is gauged before any of them
// reports, so a machine missing several is told about all of them at once.
func (foundry *Foundry) Gauge(ctx context.Context, machineries []v1alpha1.Machinery) error {
	proven := map[string]struct{}{}
	unavailable := []string{}

	for _, machinery := range machineries {
		p, err := foundry.Plan(ctx, machinery)
		if err != nil {
			return err
		}

		foundry.Logger.InfoContext(ctx, "gauging",
			slog.String("casting.kind", machinery.Kind().String()),
			slog.String("casting.metadata.name", machinery.Name()))

		for _, tool := range p.Toolers() {
			if _, done := proven[tool.Name()]; done {
				continue
			}

			proven[tool.Name()] = struct{}{}

			if err := tool.Gauge(ctx); err != nil {
				foundry.Logger.ErrorContext(ctx, "tool is not available or cannot be detected properly",
					slog.String("tool.name", tool.Name()), foundryerrors.LogAttr(err))
				unavailable = append(unavailable, tool.Name())

				continue
			}

			foundry.Logger.InfoContext(ctx, "tool is available", slog.String("tool.name", tool.Name()))
		}
	}

	if len(unavailable) > 0 {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "tools are not available, please install them and try again: %s", strings.Join(unavailable, ", "))
	}

	return nil
}
