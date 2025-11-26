package main

import (
	"log/slog"

	"github.com/SigNoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
)

func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "gauge"))
			ctx := cmd.Context()

			logger.DebugContext(ctx, "starting command", slog.String("cfg.file", cfg.File))
			return nil
		},
	}

	rootCmd.AddCommand(gaugeCmd)
}
