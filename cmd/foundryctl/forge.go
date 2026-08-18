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
		RunE: recoverRunE(domain.EventForge, func(cmd *cobra.Command, args []string) ([]domain.Properties, error) {
			return runForge(cmd.Context(), rootLogger, commonCfg.File, poursCfg.Path)
		}),
	}

	rootCmd.AddCommand(forgeCmd)
}

func runForge(ctx context.Context, logger *slog.Logger, path string, poursPath string) ([]domain.Properties, error) {
	foundry, err := foundry.New(logger)
	if err != nil {
		return nil, err
	}

	machineries, err := foundry.Config.GetV1Alpha1(ctx, path)
	if err != nil {
		return nil, err
	}

	kinds := kindsOf(machineries)

	poursAbsPath, err := filepath.Abs(poursPath)
	if err != nil {
		return []domain.Properties{domain.NewProperties().Set("kinds", kinds).WithError(err)}, err
	}

	props := []domain.Properties{}
	for _, machinery := range machineries {
		if err := foundry.Forge(ctx, machinery, path, &writer.Options{Output: &os.File{}, TargetDirectory: poursAbsPath}); err != nil {
			return append(props, domain.NewProperties().Set("kind", machinery.Kind().String()).Set("kinds", kinds).WithError(err)), err
		}

		props = append(props, machinery.TrackableProperties().Set("kinds", kinds).WithSuccess())
	}

	return props, nil
}
