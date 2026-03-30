// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/ledger"
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(commonCfg.Debug)
			tracker := newTracker()
			defer func() {
				_ = tracker.Close()
			}()

			if !castCfg.NoGauge {
				err := runGauge(ctx, logger, tracker, commonCfg.File)
				if err != nil {
					return err
				}
			}

			if !castCfg.NoForge {
				err := runForge(ctx, logger, tracker, commonCfg.File, poursCfg.Path)
				if err != nil {
					return err
				}
			}

			return runCast(ctx, logger, tracker, poursCfg.Path, commonCfg.File)
		},
	}

	rootCmd.AddCommand(castCmd)
	castCfg.RegisterFlags(castCmd)
}

func runCast(ctx context.Context, logger *slog.Logger, tracker ledger.Ledger, poursPath string, configPath string) error {
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

	lock, err := foundry.Config.GetV1Alpha1Lock(ctx, configPath)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load generated casting.yaml.lock", foundryerrors.LogAttr(err))
		tracker.Track(ctx, ledger.WithError(ledger.CommandProperties("cast"), err))
		return err
	}

	props := ledger.CastingProperties("cast", lock)

	err = foundry.Cast(ctx, lock, poursPath)
	if err != nil {
		tracker.Track(ctx, ledger.WithError(props, err))
		return err
	}

	tracker.Track(ctx, ledger.WithSuccess(props))
	return nil
}
