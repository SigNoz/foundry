package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/spf13/cobra"
)

func registerUncastCmd(rootCmd *cobra.Command) {
	uncastCmd := &cobra.Command{
		Use:   "uncast",
		Short: "Remove the cast deployment. Definitions are removed; data is never touched.",
		RunE: recoverRunE(domain.EventUncast, func(cmd *cobra.Command, args []string, report reporter) error {
			ctx := cmd.Context()

			if !uncastCfg.Yes {
				return errors.Newf(errors.TypeInvalidInput, "uncast removes the deployment (data and volumes always stay); re-run with --yes to confirm")
			}

			return runUncast(ctx, rootLogger, poursCfg.Path, commonCfg.File, report)
		}),
	}

	rootCmd.AddCommand(uncastCmd)
	uncastCfg.RegisterFlags(uncastCmd)
}

func runUncast(ctx context.Context, logger *slog.Logger, poursPath string, configPath string, report reporter) error {
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

	// Backwards against the order the lock records, so a workload leaves
	// before the substrate it runs on.
	for _, machinery := range slices.Backward(machineries) {
		if err := foundry.Uncast(ctx, machinery, poursPath); err != nil {
			report(machinery.TrackableProperties(), err)

			return err
		}

		report(machinery.TrackableProperties(), nil)
	}

	return nil
}
