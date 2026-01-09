package main

import (
	"context"
	"fmt"
	"log/slog"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/loader/yamlloader"
	"github.com/spf13/cobra"
)

func registerForgeCmd(rootCmd *cobra.Command) {
	var outputDir string

	forgeCmd := &cobra.Command{
		Use:   "forge",
		Short: "Forge Configuration and Deployment Files",
		Long:  "Generate deployment configuration files from casting.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(cfg.Debug)

			return runForge(ctx, logger, cfg.File)
		},
	}

	forgeCmd.Flags().StringVarP(&outputDir, "output", "o", "./pours", "Output Directory for pours containing the deployment and configuration files")
	rootCmd.AddCommand(forgeCmd)
}

func runForge(ctx context.Context, logger *slog.Logger, path string) error {
	yamlLoader := yamlloader.New()

	config, err := yamlLoader.LoadV1Alpha1(ctx, path)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load casting", foundryerrors.LogAttr(err))
		return err
	}

	castings := foundry.NewCastings(logger)
	casting, ok := castings[config.Spec.Deployment.Mode]
	if !ok {
		return fmt.Errorf("deployment mode '%s' is not supported", config.Spec.Deployment.Mode)
	}

	_, err = casting.Forge(ctx, config)
	if err != nil {
		logger.ErrorContext(ctx, "failed to forge casting", foundryerrors.LogAttr(err))
		return err
	}

	return nil
}
