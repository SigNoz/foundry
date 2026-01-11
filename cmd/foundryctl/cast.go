// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"context"
	"log/slog"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(cfg.Debug)

			return runCast(ctx, logger, cfg.File, out.Path)
		},
	}

	rootCmd.AddCommand(castCmd)
}

func runCast(ctx context.Context, logger *slog.Logger, path string, outputPath string) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	casting, err := foundry.Loader.LoadV1Alpha1(ctx, path)
	if err != nil {
		logger.ErrorContext(ctx, err.Error())
		return err
	}

	err = foundry.Cast(ctx, casting, outputPath)
	if err != nil {
		logger.ErrorContext(ctx, err.Error())
		return err
	}

	return nil
}
