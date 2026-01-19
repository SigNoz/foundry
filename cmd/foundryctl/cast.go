// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"context"
	"log/slog"
	"os"

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

			return runCast(ctx, logger, cfg.File, out.Path, pours.Path)
		},
	}

	rootCmd.AddCommand(castCmd)
}

func runCast(ctx context.Context, logger *slog.Logger, path string, outputPath string, poursPathFlag string) error {
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

	// Determine the pours path to use
	var poursPath string

	if poursPathFlag != "" {
		// User explicitly provided --pours flag - assume files are already generated
		poursPath = poursPathFlag
		logger.InfoContext(ctx, "Using existing pours directory", slog.String("poursPath", poursPath))
	} else {
		// No --pours flag provided - generate pours first using Forge
		logger.InfoContext(ctx, "Pours path not provided, generating pours first", slog.String("outputPath", outputPath))
		err = foundry.Forge(ctx, casting, &writer.Options{
			Output:          &os.File{},
			TargetDirectory: outputPath,
		})
		if err != nil {
			logger.ErrorContext(ctx, "Failed to forge configuration", foundryerrors.LogAttr(err))
			return err
		}
		poursPath = outputPath
		logger.InfoContext(ctx, "Successfully generated pours", slog.String("poursPath", poursPath))
	}

	// Now cast using the pours directory
	err = foundry.Cast(ctx, casting, poursPath)
	if err != nil {
		logger.ErrorContext(ctx, err.Error())
		return err
	}

	return nil
}
