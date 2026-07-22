package foundry

import (
	"context"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

func (foundry *Foundry) Gauge(ctx context.Context, machinery v1alpha1.Machinery) error {
	p, err := foundry.newPlanner(ctx, machinery)
	if err != nil {
		return err
	}

	unavailableTools := []string{}
	for _, tooler := range p.Toolers() {
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
