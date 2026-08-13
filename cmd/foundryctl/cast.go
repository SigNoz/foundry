// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/writer"
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: recoverRunE(domain.EventCast, func(cmd *cobra.Command, args []string) ([]domain.Properties, error) {
			return runCast(cmd.Context(), rootLogger, poursCfg.Path, commonCfg.File)
		}),
	}

	rootCmd.AddCommand(castCmd)
	castCfg.RegisterFlags(castCmd)
}

// runCast gauges, forges, and casts one set of castings. Forge resolves the set
// in place and records it in the lock, so cast applies what was just recorded.
// Without forge nothing resolves the castings, so the lock is the only source.
func runCast(ctx context.Context, logger *slog.Logger, poursPath string, configPath string) ([]domain.Properties, error) {
	foundry, err := foundry.New(logger)
	if err != nil {
		return nil, err
	}

	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to resolve pours path")
	}

	read := foundry.Config.GetV1Alpha1
	if castCfg.NoForge {
		read = foundry.Config.GetV1Alpha1Lock
	}

	machineries, err := read(ctx, configPath)
	if err != nil {
		return nil, err
	}

	props := trackableProperties(machineries)

	planners, err := foundry.Plan(ctx, machineries)
	if err != nil {
		return props, err
	}

	if !castCfg.NoGauge {
		if err := foundry.Gauge(ctx, planners); err != nil {
			return props, err
		}
	}

	if !castCfg.NoForge {
		if err := foundry.Forge(ctx, planners, configPath, &writer.Options{Output: &os.File{}, TargetDirectory: poursPath}); err != nil {
			return props, err
		}
	}

	err = foundry.Cast(ctx, planners, poursPath)
	return props, err
}
