package foundry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
)

func (foundry *Foundry) Gauge(ctx context.Context, machinery v1alpha1.Machinery) error {
	switch c := machinery.(type) {
	case *installation.Casting:
		return foundry.gaugeInstallation(ctx, *c)
	case *collectionagent.Casting:
		return foundry.gaugeCollectionAgent(ctx, *c)
	}
	return fmt.Errorf("unsupported casting kind %q", machinery.Kind())
}

func (foundry *Foundry) gaugeCollectionAgent(ctx context.Context, config collectionagent.Casting) error {
	foundry.Logger.WarnContext(ctx, "collectionagent gauge not yet implemented; skipping",
		slog.String("casting.metadata.name", config.Metadata.Name))
	return nil
}

func (foundry *Foundry) gaugeInstallation(ctx context.Context, config installation.Casting) error {
	foundry.Logger.InfoContext(ctx, "starting gauge pipeline", slog.String("casting.metadata.name", config.Metadata.Name))

	spec := &config.Spec

	toolers, err := foundry.Registry.Toolers(spec.Deployment)
	if err != nil {
		return err
	}

	if spec.Infrastructure.Enabled {
		toolers = append(toolers, terraformtooler.New())
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
