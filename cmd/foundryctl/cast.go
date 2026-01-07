package main

import (
	"log/slog"

	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "cast"))
			ctx := cmd.Context()

			logger.DebugContext(ctx, "starting command", slog.String("cfg.file", cfg.File))
			return nil
		},
	}

	rootCmd.AddCommand(castCmd)
}
