package main

import (
	"context"
	"log/slog"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/ledger"
	"github.com/spf13/cobra"
)

func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(commonCfg.Debug)
			tracker := newTracker()
			defer func() {
				_ = tracker.Close()
			}()

			return runGauge(ctx, logger, tracker, commonCfg.File)
		},
	}

	rootCmd.AddCommand(gaugeCmd)
}

func runGauge(ctx context.Context, logger *slog.Logger, tracker ledger.Ledger, path string) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	casting, err := foundry.Config.GetV1Alpha1(ctx, path)
	if err != nil {
		logger.ErrorContext(ctx, err.Error())
		tracker.Track(ctx, ledger.WithError(ledger.CommandProperties("gauge"), err))
		return err
	}

	props := ledger.CastingProperties("gauge", casting)

	err = foundry.Gauge(ctx, casting)
	if err != nil {
		logger.ErrorContext(ctx, err.Error())
		tracker.Track(ctx, ledger.WithError(props, err))
		return err
	}

	tracker.Track(ctx, ledger.WithSuccess(props))
	return nil
}
