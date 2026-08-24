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

func registerMeltCmd(rootCmd *cobra.Command) {
	meltCmd := &cobra.Command{
		Use:   "melt",
		Short: "Remove the cast deployment",
		Long:  "Remove the cast deployment from the target environment; data is never touched",
		RunE: recoverRunE(domain.EventMelt, func(cmd *cobra.Command, args []string) (domain.Properties, error) {
			ctx := cmd.Context()

			if meltCfg.Yes {
				ctx = tooler.WithApproval(ctx)
			}

			return runMelt(ctx, rootLogger, poursCfg.Path, commonCfg.File)
		}),
	}

	rootCmd.AddCommand(meltCmd)
	meltCfg.RegisterFlags(meltCmd)
}

func runMelt(ctx context.Context, logger *slog.Logger, poursPath string, configPath string) (domain.Properties, error) {
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

	err = foundry.Melt(ctx, machineries, poursPath)
	return props, err
}
