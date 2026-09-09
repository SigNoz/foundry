// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: recoverRunE(domain.EventCast, func(cmd *cobra.Command, args []string, report reporter) error {
			ctx := cmd.Context()

			// A document the inner stages pass is prepared, not cast, so
			// only their failures report.
			prepReport := reporter(func(props domain.Properties, err error) {
				if err != nil {
					report(props, err)
				}
			})

			if !castCfg.NoGauge {
				if err := runGauge(ctx, rootLogger, commonCfg.File, prepReport); err != nil {
					return err
				}
			}

			if !castCfg.NoForge {
				if err := runForge(ctx, rootLogger, commonCfg.File, poursCfg.Path, prepReport); err != nil {
					return err
				}
			}

			return runCast(ctx, rootLogger, poursCfg.Path, commonCfg.File, report)
		}),
	}

	rootCmd.AddCommand(castCmd)
	castCfg.RegisterFlags(castCmd)
}

func runCast(ctx context.Context, logger *slog.Logger, poursPath string, configPath string, report reporter) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		return err
	}

	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to resolve pours path")
	}

	machineries, err := foundry.Config.GetV1Alpha1Lock(ctx, configPath)
	if err != nil {
		return err
	}

	for _, machinery := range machineries {
		if err := foundry.Cast(ctx, machinery, poursPath); err != nil {
			report(machinery.TrackableProperties(), err)
			return err
		}

		report(machinery.TrackableProperties(), nil)
	}

	return nil
}
