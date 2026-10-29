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
		Args:  cobra.NoArgs,
		RunE:  recoverRunE(domain.EventCast, castUp),
	}

	registerCastUpCmd(castCmd)
	registerCastDownCmd(castCmd)

	rootCmd.AddCommand(castCmd)
	castCfg.RegisterFlags(castCmd)
}

func registerCastUpCmd(castCmd *cobra.Command) {
	castUpCmd := &cobra.Command{
		Use:   "up",
		Short: "Cast to the target environment (same as cast).",
		Args:  cobra.NoArgs,
		RunE:  recoverRunE(domain.EventCast, castUp),
	}

	castCmd.AddCommand(castUpCmd)
}

func registerCastDownCmd(castCmd *cobra.Command) {
	castDownCmd := &cobra.Command{
		Use:   "down",
		Short: "Take down what cast deployed. Volumes and data stay.",
		Args:  cobra.NoArgs,
		RunE: recoverRunE(domain.EventCastDown, func(cmd *cobra.Command, args []string) (domain.Properties, error) {
			return runDown(cmd.Context(), rootLogger, poursCfg.Path, commonCfg.File)
		}),
	}

	castCmd.AddCommand(castDownCmd)
}

func castUp(cmd *cobra.Command, args []string) (domain.Properties, error) {
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
}

func runCast(ctx context.Context, logger *slog.Logger, poursPath string, configPath string) (domain.Properties, error) {
	foundry, err := foundry.New(logger)
	if err != nil {
		return domain.NewProperties(), err
	}

	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return domain.NewProperties(), errors.Wrapf(err, errors.TypeInternal, "failed to resolve pours path")
	}

	machineries, err := foundry.Config.GetV1Alpha1Lock(ctx, configPath)
	if err != nil {
		return domain.NewProperties(), err
	}

	props := domain.NewProperties()
	for _, machinery := range machineries {
		props = props.Merge(machinery.TrackableProperties())
	}

	n, err := foundry.Cast(ctx, machineries, poursPath)
	if n < len(machineries) {
		props = props.Merge(machineries[n].FailedProperties())
	}

	return props, err
}

func runDown(ctx context.Context, logger *slog.Logger, poursPath string, configPath string) (domain.Properties, error) {
	foundry, err := foundry.New(logger)
	if err != nil {
		return domain.NewProperties(), err
	}

	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return domain.NewProperties(), errors.Wrapf(err, errors.TypeInternal, "failed to resolve pours path")
	}

	machineries, err := foundry.Config.GetV1Alpha1(ctx, configPath)
	if err != nil {
		return domain.NewProperties(), err
	}

	props := domain.NewProperties()
	for _, machinery := range machineries {
		props = props.Merge(machinery.TrackableProperties())
	}

	if !castCfg.NoGauge {
		n, err := foundry.Gauge(ctx, machineries)
		if n < len(machineries) {
			props = props.Merge(machineries[n].FailedProperties())
		}

		if err != nil {
			return props, err
		}
	}

	n, err := foundry.Down(ctx, machineries, poursPath)
	if n < len(machineries) {
		props = props.Merge(machineries[len(machineries)-1-n].FailedProperties())
	}

	return props, err
}
