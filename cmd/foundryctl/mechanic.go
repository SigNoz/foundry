package main

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/mechanic"
	"github.com/spf13/cobra"
)

func registerMechanicCmd(rootCmd *cobra.Command) {
	mechanicCmd := &cobra.Command{
		Use:   "mechanic",
		Short: "Diagnose a running deployment.",
	}

	registerMechanicInspectCmd(mechanicCmd)

	rootCmd.AddCommand(mechanicCmd)
}

func registerMechanicInspectCmd(mechanicCmd *cobra.Command) {
	inspectCmd := &cobra.Command{
		Use:   "inspect <molding> <entity-kind> <entity-id>",
		Short: "Inspect a named entity within a deployment.",
		Long: `Inspect a named entity within a deployment.

The resource path accepts both slash and positional forms, with an optional
leading casting kind (resolved from the lock file when omitted):

  foundryctl mechanic inspect signoz alert <id>
  foundryctl mechanic inspect signoz/alert/<id>
  foundryctl mechanic inspect installation signoz alert <id>
  foundryctl mechanic inspect telemetrystore table <name>

When the deployment was not provisioned by foundry, supply connection details
via flags (or their environment variables) to override the lock file.`,
		Args: cobra.MinimumNArgs(1),
		RunE: recoverRunE(domain.EventMechanic, func(cmd *cobra.Command, args []string) (domain.Properties, error) {
			return runMechanicInspect(cmd.Context(), rootLogger, commonCfg.File, args)
		}),
	}

	mechanicCfg.RegisterFlags(inspectCmd)
	mechanicCmd.AddCommand(inspectCmd)
}

func runMechanicInspect(ctx context.Context, logger *slog.Logger, configPath string, args []string) (domain.Properties, error) {
	resource, err := mechanic.ParseResource(args)
	if err != nil {
		return domain.NewProperties(), err
	}

	f, err := foundry.New(logger)
	if err != nil {
		return domain.NewProperties(), err
	}

	machinery, err := f.Config.GetV1Alpha1Lock(ctx, configPath)
	if err != nil {
		return domain.NewProperties(), err
	}

	props := machinery.TrackableProperties()

	overrides := mechanic.Overrides{
		Signoz:        mechanicCfg.Signoz,
		ClickhouseDSN: mechanicCfg.ClickhouseDSN,
		MetastoreDSN:  mechanicCfg.MetastoreDSN,
	}

	err = f.Inspect(ctx, machinery, resource, overrides)
	return props, err
}
