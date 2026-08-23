package main

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/spf13/cobra"
)

func registerUncastCmd(rootCmd *cobra.Command) {
	uncastCmd := &cobra.Command{
		Use:   "uncast",
		Short: "Remove the cast deployment. Definitions are removed; data is never touched.",
		RunE: recoverRunE(domain.EventUncast, func(cmd *cobra.Command, args []string) (domain.Properties, error) {
			ctx := cmd.Context()

			if uncastCfg.Yes {
				ctx = tooler.WithApproval(ctx)
			}

			return runUncast(ctx, rootLogger, poursCfg.Path, commonCfg.File)
		}),
	}

	rootCmd.AddCommand(uncastCmd)
	uncastCfg.RegisterFlags(uncastCmd)
}

func runUncast(ctx context.Context, logger *slog.Logger, poursPath string, configPath string) (domain.Properties, error) {
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
		if machinery.Kind() == v1alpha1.KindInstallation {
			props = machinery.TrackableProperties()
		}
	}

	err = foundry.Uncast(ctx, machineries, poursPath)
	return props, err
}
