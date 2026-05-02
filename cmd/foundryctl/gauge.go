package main

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/spf13/cobra"
)

func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE: recoverRunE(domain.EventGauge, func(cmd *cobra.Command, args []string) (domain.Properties, error) {
			return runGauge(cmd.Context(), rootLogger, commonCfg.File)
		}),
	}

	rootCmd.AddCommand(gaugeCmd)
}

func runGauge(ctx context.Context, logger *slog.Logger, path string) (domain.Properties, error) {
	foundry, err := foundry.New(logger)
	if err != nil {
		return domain.NewProperties(), err
	}

	casting, err := foundry.Config.GetV1Alpha1(ctx, path)
	if err != nil {
		return domain.NewProperties(), err
	}

	props := casting.TrackableProperties()

	err = foundry.Gauge(ctx, casting)
	return props, err
}
