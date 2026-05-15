// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: recoverRunE(domain.EventCast, func(cmd *cobra.Command, args []string) (domain.Properties, error) {
			ctx := cmd.Context()

			if !castCfg.NoGauge {
				if props, err := runGauge(ctx, rootLogger, commonCfg.File); err != nil {
					return props, err
				}
			}

			if !castCfg.NoForge {
				if props, err := runForge(ctx, rootLogger, commonCfg.File, poursCfg.Path); err != nil {
					return props, err
				}
			}

			return runCast(ctx, rootLogger, poursCfg.Path, commonCfg.File)
		}),
	}

	rootCmd.AddCommand(castCmd)
	castCfg.RegisterFlags(castCmd)
}

func runCast(ctx context.Context, logger *slog.Logger, poursPath string, configPath string) (domain.Properties, error) {
	foundry, err := foundry.New(logger)
	if err != nil {
		return domain.NewProperties(), err
	}

	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return domain.NewProperties(), fmt.Errorf("failed to resolve pours path: %w", err)
	}

	machinery, err := foundry.Config.GetV1Alpha1Lock(ctx, configPath)
	if err != nil {
		return domain.NewProperties(), err
	}

	props := machinery.TrackableProperties()

	err = foundry.Cast(ctx, machinery, poursPath)
	return props, err
}
