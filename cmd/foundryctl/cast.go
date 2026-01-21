// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/writer"
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(cfg.Debug)

			return runCast(ctx, logger, cfg.File, pours.Path)
		},
	}

	rootCmd.AddCommand(castCmd)
}

func runCast(ctx context.Context, logger *slog.Logger, configPath string, poursPath string) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	// Get absolute pours path
	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return fmt.Errorf("failed to resolve pours path: %w", err)
	}

	// Check if casting.yaml.lock exists in pours directory (already forged)
	castingLock := filepath.Join(poursPath, "casting.yaml.lock")
	if _, err := os.Stat(castingLock); err == nil {
		// Already forged - load from lock file and cast directly
		logger.InfoContext(ctx, "Found existing casting.yaml.lock, skipping forge", slog.String("pours_path", poursPath))

		casting, err := foundry.Loader.LoadV1Alpha1(ctx, castingLock)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to load casting.yaml.lock", foundryerrors.LogAttr(err))
			return err
		}

		return foundry.Cast(ctx, casting, poursPath)
	}

	// Not forged yet - load config, forge, then cast
	logger.InfoContext(ctx, "No casting.yaml.lock found, forging first", slog.String("pours_path", poursPath))

	casting, err := foundry.Loader.LoadV1Alpha1(ctx, configPath)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to load casting config", foundryerrors.LogAttr(err))
		return err
	}

	// Forge the configuration
	if err := foundry.Forge(ctx, casting, &writer.Options{
		Output:          &os.File{},
		TargetDirectory: poursPath,
	}); err != nil {
		logger.ErrorContext(ctx, "Failed to forge configuration", foundryerrors.LogAttr(err))
		return err
	}

	// Reload from the generated lock file
	casting, err = foundry.Loader.LoadV1Alpha1(ctx, castingLock)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to load generated casting.yaml.lock", foundryerrors.LogAttr(err))
		return err
	}

	return foundry.Cast(ctx, casting, poursPath)
}
