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

			// The inner stages report under the cast event.
			if !castCfg.NoGauge {
				if props, err := runGauge(ctx, rootLogger, commonCfg.File); err != nil {
					return props, err
				}
			}

			// What the forge stage forged was not cast, so only its failure
			// reports.
			if !castCfg.NoForge {
				if props, err := runForge(ctx, rootLogger, commonCfg.File, poursCfg.Path); err != nil {
					failed := []domain.Properties{}
					for _, p := range props {
						if p.Succeeded() {
							continue
						}
						failed = append(failed, p)
					}
					return failed, err
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
			return append(props, domain.NewProperties().Set("kind", machinery.Kind().String()).Set("kinds", kinds).WithError(err)), err
		}

		props = append(props, machinery.TrackableProperties().Set("kinds", kinds).WithSuccess())
	}

	return props, nil
}
