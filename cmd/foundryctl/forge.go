package main

import (
	"log/slog"

	"github.com/SigNoz/foundry/internal/instrumentation"
	"github.com/SigNoz/foundry/internal/loader"
	"github.com/SigNoz/foundry/internal/output"
	"github.com/SigNoz/foundry/internal/registry"
	"github.com/spf13/cobra"
)

func registerForgeCmd(rootCmd *cobra.Command) {
	
	var outputDir string

	forgeCmd := &cobra.Command{
	Use: "forge",
	Short: "Forge Configuration and Deployment Files",
	Long: "Generate deployment configuration files from casting.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
			
		ctx := cmd.Context()
		logger := instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "forge"))
			
		config, err := loader.LoadConfig(cfg.File)
		if err != nil {
			logger.ErrorContext(ctx, "config load failed", slog.String("error", err.Error()))
			return err
		}
			
		logger.DebugContext(ctx, "Configuration loaded",
			slog.String("platform", config.Platform),
			slog.Int("enabled_components", len(config.EnabledComponents)))

		outputMgr, err := output.NewManager(outputDir)
		if err != nil {
			logger.ErrorContext(ctx, "output manager init failed", slog.Any("error", err))
			return err
		}

		factory, err := registry.NewFactory(config)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create registry factory", slog.Any("error", err))
			return err
		}
		componentRegistry := factory.CreateComponentRegistry()

		allFiles, err := componentRegistry.GenerateAllEnabled()
		if err != nil {
			logger.ErrorContext(ctx, "failed to generate components", slog.Any("error", err))
			return err
		}

		// Write all generated files
		for componentID, files := range allFiles {
			componentName := string(componentID)
			logger.DebugContext(ctx, "✓ Component generated", slog.String("component", componentName),slog.Int("files", len(files)))
				
			// Write files to output manager
			if err := outputMgr.WriteComponent(componentName, files); err != nil {
				logger.ErrorContext(ctx, "write component failed", slog.String("component", componentName), slog.Any("error", err))
				return err
			}
		}

		logger.InfoContext(ctx, "✓ Successfully forged configuration", slog.String("output", outputDir), slog.String("platform", config.Platform), slog.Int("components", len(allFiles)))

		return nil
	},
}
	
	forgeCmd.Flags().StringVarP(&outputDir, "output", "o", "./pours", "Output Directory for pours containing the deployment and configuration files")
	rootCmd.AddCommand(forgeCmd)
}