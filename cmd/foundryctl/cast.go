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
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(cfg.Debug)

			return runCast(ctx, logger, pours.Path)
		},
	}

	rootCmd.AddCommand(castCmd)
}

func runCast(ctx context.Context, logger *slog.Logger, poursPath string) error {
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

	castingLock := filepath.Join(poursPath, "casting.yaml.lock")
	if _, err := os.Stat(castingLock); err != nil {
		return fmt.Errorf("casting.yaml.lock does not exist at given pours path: %s. Please run forge before cast", poursPath)
	}

	// Load casting from the generated lock file
	casting, err := foundry.Loader.LoadV1Alpha1(ctx, castingLock)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load generated casting.yaml.lock", foundryerrors.LogAttr(err))
		return err
	}

	return foundry.Cast(ctx, casting, poursPath)
}
