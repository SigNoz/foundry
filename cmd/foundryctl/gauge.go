package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/loader/yamlloader"
	"github.com/spf13/cobra"
)

func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(cfg.Debug)

			return runGauge(ctx, logger, cfg.File)
		},
	}

	rootCmd.AddCommand(gaugeCmd)
}

func runGauge(ctx context.Context, logger *slog.Logger, path string) error {
	yamlLoader := yamlloader.New()

	casting, err := yamlLoader.LoadV1Alpha1(ctx, path)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load casting", foundryerrors.LogAttr(err))
		return err
	}

	deploymentMode := casting.Spec.Deployment.Mode
	toolers, ok := foundry.DeploymentModeToTooler[deploymentMode]
	if !ok {
		err := fmt.Errorf("deployment mode '%s' is not yet supported. Raise an issue at https://github.com/signoz/foundry/issues to request support for this mode.", deploymentMode)
		return err
	}

	errTools := []string{}

	for _, tooler := range toolers {
		err := tooler.Gauge(ctx)
		if err != nil {
			logger.ErrorContext(ctx, "tool '%s' not found", tooler.Name(), foundryerrors.LogAttr(err))
			errTools = append(errTools, tooler.Name())
			continue
		}

		logger.InfoContext(ctx, "tool is available", slog.String("tool.name", tooler.Name()))
	}

	if len(errTools) > 0 {
		return fmt.Errorf("tools are not available, please install them and try again: %s", strings.Join(errTools, ", "))
	}

	return nil
}
