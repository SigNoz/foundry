// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"context"
	"fmt"
	"path/filepath"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/ux"
	"github.com/spf13/cobra"
)

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			u := ux.New(commonCfg.Debug)

			if !castCfg.NoGauge {
				err := runGauge(ctx, u, commonCfg.File)
				if err != nil {
					return err
				}
			}

			if !castCfg.NoForge {
				err := runForge(ctx, u, commonCfg.File, poursCfg.Path)
				if err != nil {
					return err
				}
			}

			return runCast(ctx, u, poursCfg.Path, commonCfg.File)
		},
	}

	rootCmd.AddCommand(castCmd)
	castCfg.RegisterFlags(castCmd)
}

func runCast(ctx context.Context, u *ux.UX, poursPath string, configPath string) error {
	f, err := foundry.New(u.Logger(), u)
	if err != nil {
		u.Logger().ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	// Get absolute pours path
	poursPath, err = filepath.Abs(poursPath)
	if err != nil {
		return fmt.Errorf("failed to resolve pours path: %w", err)
	}

	lock, err := f.Config.GetV1Alpha1Lock(ctx, configPath)
	if err != nil {
		u.Logger().ErrorContext(ctx, "failed to load generated casting.yaml.lock", foundryerrors.LogAttr(err))
		return err
	}

	return f.Cast(ctx, lock, poursPath)
}
