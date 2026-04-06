package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/ledger"
	"github.com/signoz/foundry/internal/writer"
	"github.com/spf13/cobra"
)

func registerForgeCmd(rootCmd *cobra.Command) {
	forgeCmd := &cobra.Command{
		Use:   "forge",
		Short: "Forge Configuration and Deployment Files",
		Long:  "Generate deployment configuration files from casting.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(commonCfg.Debug)
			tracker := newTracker()
			defer func() {
				_ = tracker.Close()
			}()

			return runForge(ctx, logger, tracker, commonCfg.File, poursCfg.Path)
		},
	}

	rootCmd.AddCommand(forgeCmd)
}

func runForge(ctx context.Context, logger *slog.Logger, tracker ledger.Ledger, path string, poursPath string) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	config, err := foundry.Config.GetV1Alpha1(ctx, path)
	if err != nil {
		tracker.Track(ctx, ledger.EventForge, ledger.WithError(nil, err))
		return err
	}

	props := ledger.CastingProperties(config)

	poursAbsPath, err := filepath.Abs(poursPath)
	if err != nil {
		tracker.Track(ctx, ledger.EventForge, ledger.WithError(props, err))
		return err
	}

	err = foundry.Forge(ctx, config, path, &writer.Options{Output: &os.File{}, TargetDirectory: poursAbsPath})
	if err != nil {
		tracker.Track(ctx, ledger.EventForge, ledger.WithError(props, err))
		return err
	}

	tracker.Track(ctx, ledger.EventForge, ledger.WithSuccess(props))
	return nil
}
