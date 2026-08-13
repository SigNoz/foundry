package foundry

import (
	"context"
	"log/slog"
	"strings"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/planner"
	"github.com/signoz/foundry/internal/tooler"
)

// Gauge checks the tools the whole casting file needs. Documents that share a
// tool gauge it once, so a machine is neither probed nor reported twice.
func (foundry *Foundry) Gauge(ctx context.Context, planners []planner.Planner) error {
	toolers := []tooler.Tooler{}
	for _, p := range planners {
		toolers = append(toolers, p.Toolers()...)
	}

	unavailableTools := []string{}
	for _, tooler := range dedupeByName(toolers) {
		if err := tooler.Gauge(ctx); err != nil {
			foundry.Logger.ErrorContext(ctx, "tool is not available or cannot be detected properly", slog.String("tool.name", tooler.Name()), foundryerrors.LogAttr(err))
			unavailableTools = append(unavailableTools, tooler.Name())
			continue
		}
		foundry.Logger.InfoContext(ctx, "tool is available", slog.String("tool.name", tooler.Name()))
	}
	if len(unavailableTools) > 0 {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "tools are not available, please install them and try again: %s", strings.Join(unavailableTools, ", "))
	}
	return nil
}

// dedupeByName keeps the first tooler of each name, in the order the documents
// asked for them.
func dedupeByName(toolers []tooler.Tooler) []tooler.Tooler {
	deduped := make([]tooler.Tooler, 0, len(toolers))
	named := make(map[string]struct{}, len(toolers))

	for _, tooler := range toolers {
		if _, gathered := named[tooler.Name()]; gathered {
			continue
		}

		named[tooler.Name()] = struct{}{}
		deduped = append(deduped, tooler)
	}

	return deduped
}
