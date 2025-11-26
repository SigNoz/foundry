package main

import (
	"log/slog"

	"github.com/SigNoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
)

func registerForgeCmd(rootCmd *cobra.Command) {
	forgeCmd := &cobra.Command{
		Use:   "forge",
		Short: "Forge configuration files from moldings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "forge"))
			ctx := cmd.Context()

			logger.DebugContext(ctx, "starting command", slog.String("cfg.file", cfg.File))
			return nil
		},
	}

	rootCmd.AddCommand(forgeCmd)
}
