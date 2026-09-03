package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"

	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/spf13/cobra"
)

func registerMeltCmd(rootCmd *cobra.Command) {
	meltCmd := &cobra.Command{
		Use:   "melt",
		Short: "Remove the cast deployment",
		Long:  "Remove the cast deployment from the target environment; data is never touched",
		RunE: recoverRunE(domain.EventMelt, func(cmd *cobra.Command, args []string, report reporter) error {
			ctx := cmd.Context()

			if meltCfg.Yes {
				ctx = tooler.WithApproval(ctx)
			}

			return runMelt(ctx, rootLogger, poursCfg.Path, commonCfg.File, report)
		}),
	}

	rootCmd.AddCommand(meltCmd)
	meltCfg.RegisterFlags(meltCmd)
}

func runMelt(ctx context.Context, logger *slog.Logger, poursPath string, configPath string, report reporter) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		return err
	}

	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to resolve pours path")
	}

	machineries, err := foundry.Config.GetV1Alpha1Lock(ctx, configPath)
	if err != nil {
		return err
	}

	// Backwards against the order the lock records, so a workload leaves
	// before the substrate it runs on.
	for _, machinery := range slices.Backward(machineries) {
		if err := foundry.Melt(ctx, machinery, poursPath); err != nil {
			report(machinery.TrackableProperties(), err)

			return err
		}

		report(machinery.TrackableProperties(), nil)
	}

	return nil
}
