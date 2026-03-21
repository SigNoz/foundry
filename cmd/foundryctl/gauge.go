package main

import (
	"context"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/ux"
	"github.com/spf13/cobra"
)

func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			u := ux.New(commonCfg.Debug)

			return runGauge(ctx, u, commonCfg.File)
		},
	}

	rootCmd.AddCommand(gaugeCmd)
}

func runGauge(ctx context.Context, u *ux.UX, path string) error {
	f, err := foundry.New(u.Logger(), u)
	if err != nil {
		u.Logger().ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	casting, err := f.Config.GetV1Alpha1(ctx, path)
	if err != nil {
		u.Logger().ErrorContext(ctx, err.Error())
		return err
	}

	err = f.Gauge(ctx, casting)
	if err != nil {
		u.Logger().ErrorContext(ctx, err.Error())
		return err
	}

	return nil
}
