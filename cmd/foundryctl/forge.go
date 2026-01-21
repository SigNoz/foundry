package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
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
			logger := instrumentation.NewLogger(cfg.Debug)

			return runForge(ctx, logger, cfg.File, pours.Path)
		},
	}

	rootCmd.AddCommand(forgeCmd)
}

func runForge(ctx context.Context, logger *slog.Logger, path string, poursPath string) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	casting, err := foundry.Loader.LoadV1Alpha1(ctx, path)
	if err != nil {
		return err
	}
	p, err := filepath.Abs(poursPath)
	if err != nil {
		return err
	}
	err = foundry.Forge(ctx, casting, &writer.Options{
		Output:          &os.File{},
		TargetDirectory: p,
	})
	if err != nil {
		return err
	}

	return nil
}
