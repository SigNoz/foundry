package foundry

import (
	"context"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

// gaugeable is what toolers and runners both expose. Runners replace toolers
// one tool at a time, so gauge checks whichever a casting is registered with.
type gaugeable interface {
	Name() string
	Gauge(ctx context.Context) error
}

// Gauge checks the tools the whole casting file needs. Documents that share a
// tool gauge it once, so a machine is neither probed nor reported twice.
func (foundry *Foundry) Gauge(ctx context.Context, machineries []v1alpha1.Machinery) error {
	planners, err := foundry.Plan(ctx, machineries)
	if err != nil {
		return err
	}

	tools := []gaugeable{}
	for _, p := range planners {
		for _, t := range p.Toolers() {
			tools = append(tools, t)
		}

		for _, r := range p.Runners() {
			tools = append(tools, r)
		}
	}

	unavailableTools := []string{}
	for _, tool := range dedupeByName(tools) {
		if err := tool.Gauge(ctx); err != nil {
			foundry.Logger.ErrorContext(ctx, "tool is not available or cannot be detected properly", slog.String("tool.name", tool.Name()), foundryerrors.LogAttr(err))
			unavailableTools = append(unavailableTools, tool.Name())
			continue
		}

		foundry.Logger.InfoContext(ctx, "tool is available", slog.String("tool.name", tool.Name()))
	}

	if len(unavailableTools) > 0 {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "tools are not available, please install them and try again: %s", strings.Join(unavailableTools, ", "))
	}

	return nil
}

// dedupeByName keeps the first tool of each name, in the order the documents
// asked for them.
func dedupeByName(tools []gaugeable) []gaugeable {
	deduped := make([]gaugeable, 0, len(tools))
	named := make(map[string]struct{}, len(tools))

	for _, tool := range tools {
		if _, gathered := named[tool.Name()]; gathered {
			continue
		}

		named[tool.Name()] = struct{}{}
		deduped = append(deduped, tool)
	}

	return deduped
}
