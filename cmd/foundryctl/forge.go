package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/writer"
	"github.com/spf13/cobra"
)

func registerForgeCmd(rootCmd *cobra.Command) {
	forgeCmd := &cobra.Command{
		Use:   "forge",
		Short: "Forge Configuration and Deployment Files",
		Long:  "Generate deployment configuration files from casting.yaml",
		RunE: recoverRunE(domain.EventForge, func(cmd *cobra.Command, args []string, report reporter) error {
			return runForge(cmd.Context(), rootLogger, commonCfg.File, poursCfg.Path, report)
		}),
	}

	rootCmd.AddCommand(forgeCmd)
}

func runForge(ctx context.Context, logger *slog.Logger, path string, poursPath string, report reporter) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		return err
	}

	machineries, err := foundry.Config.GetV1Alpha1(ctx, path)
	if err != nil {
		return err
	}

	poursAbsPath, err := filepath.Abs(poursPath)
	if err != nil {
		return err
	}

	for _, machinery := range machineries {
		if err := foundry.Forge(ctx, machinery, path, &writer.Options{Output: &os.File{}, TargetDirectory: poursAbsPath}); err != nil {
			report(machinery.TrackableProperties(), err)
			return err
		}

		report(machinery.TrackableProperties(), nil)
	}

	return nil
}
