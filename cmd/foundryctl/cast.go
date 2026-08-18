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
		RunE: recoverRunE(domain.EventCast, func(cmd *cobra.Command, args []string) ([]domain.Properties, error) {
			ctx := cmd.Context()

			// A failed inner stage reports only its failure props under the
			// cast event: what it forged was not cast, so nothing succeeded.
			if !castCfg.NoGauge {
				if props, err := runGauge(ctx, rootLogger, commonCfg.File); err != nil {
					if len(props) > 0 {
						props = props[len(props)-1:]
					}
					return props, err
				}
			}

			if !castCfg.NoForge {
				if props, err := runForge(ctx, rootLogger, commonCfg.File, poursCfg.Path); err != nil {
					if len(props) > 0 {
						props = props[len(props)-1:]
					}
					return props, err
				}
			}

			return runCast(ctx, rootLogger, poursCfg.Path, commonCfg.File)
		}),
	}

	rootCmd.AddCommand(castCmd)
	castCfg.RegisterFlags(castCmd)
}

func runCast(ctx context.Context, logger *slog.Logger, poursPath string, configPath string) ([]domain.Properties, error) {
	foundry, err := foundry.New(logger)
	if err != nil {
		return nil, err
	}

	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to resolve pours path")
	}

	machineries, err := foundry.Config.GetV1Alpha1Lock(ctx, configPath)
	if err != nil {
		return nil, err
	}

	kinds := kindsOf(machineries)

	props := []domain.Properties{}
	for _, machinery := range machineries {
		if err := foundry.Cast(ctx, machinery, poursPath); err != nil {
			return append(props, domain.NewProperties().Set("kind", machinery.Kind().String()).Set("kinds", kinds)), err
		}

		props = append(props, machinery.TrackableProperties().Set("kinds", kinds))
	}

	return props, nil
}
