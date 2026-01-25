package foundry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

func (foundry *Foundry) Gauge(ctx context.Context, config v1alpha1.Casting) error {
	foundry.Logger.InfoContext(ctx, "starting gauge pipeline", slog.String("casting.metadata.name", config.Metadata.Name))

	toolers, err := foundry.Registry.Toolers(config.Spec.Deployment)
	if err != nil {
		return err
	}

	unavailableTools := []string{}

	for _, tooler := range toolers {
		err := tooler.Gauge(ctx)
		if err != nil {
			foundry.Logger.ErrorContext(ctx, "tool is not available or cannot be detected properly", slog.String("tool.name", tooler.Name()), foundryerrors.LogAttr(err))
			unavailableTools = append(unavailableTools, tooler.Name())
			continue
		}

		foundry.Logger.InfoContext(ctx, "tool is available", slog.String("tool.name", tooler.Name()))
	}

	if len(unavailableTools) > 0 {
		return fmt.Errorf("tools are not available, please install them and try again: %s", strings.Join(unavailableTools, ", "))
	}

	return nil
}
