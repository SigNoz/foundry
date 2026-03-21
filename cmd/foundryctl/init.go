package main

import (
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/ux"
	"github.com/signoz/foundry/internal/wizard"
	"github.com/spf13/cobra"
)

var initCfg initConfig

type initConfig struct {
	Mode           string
	Flavor         string
	Platform       string
	Name           string
	Output         string
	NonInteractive bool
}

func (c *initConfig) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Mode, "mode", "", "Deployment mode (e.g., docker, systemd, kubernetes)")
	cmd.Flags().StringVar(&c.Flavor, "flavor", "", "Deployment flavor (e.g., compose, binary, helm)")
	cmd.Flags().StringVar(&c.Platform, "platform", "", "Deployment platform (e.g., render, coolify, railway)")
	cmd.Flags().StringVar(&c.Name, "name", "", "Installation name (default: signoz)")
	cmd.Flags().StringVarP(&c.Output, "output", "o", "", "Output file path (default: casting.yaml)")
	cmd.Flags().BoolVar(&c.NonInteractive, "non-interactive", false, "Run without interactive prompts (requires --mode/--platform and --flavor)")
}

func registerInitCmd(rootCmd *cobra.Command) {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new casting.yaml configuration.",
		Long:  "Interactively create a casting.yaml file for deploying SigNoz. Use --non-interactive with flags for CI/automation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ux.New(commonCfg.Debug)

			return runInit(u)
		},
	}

	initCfg.RegisterFlags(initCmd)
	rootCmd.AddCommand(initCmd)
}

func runInit(u *ux.UX) error {
	var result *wizard.Result

	if initCfg.NonInteractive {
		r, err := wizard.BuildCastingFromFlags(initCfg.Name, initCfg.Mode, initCfg.Flavor, initCfg.Platform, initCfg.Output)
		if err != nil {
			return err
		}
		result = r
	} else {
		// Get available deployments from registry
		registry, err := foundry.NewRegistry(u.Logger())
		if err != nil {
			return err
		}

		deployments := make([]v1alpha1.TypeDeployment, 0)
		for d := range registry.CastingItems() {
			deployments = append(deployments, d)
		}

		r, err := wizard.Run(deployments)
		if err != nil {
			return err
		}
		result = r
	}

	err := wizard.WriteCasting(result)
	if err != nil {
		return fmt.Errorf("failed to write casting.yaml: %w", err)
	}

	u.Header(fmt.Sprintf("Created %s", result.OutputPath))
	u.Success(fmt.Sprintf("Deployment: %s", formatDeployment(result.Deployment)))
	u.Success(fmt.Sprintf("Name: %s", result.Name))
	u.Success(fmt.Sprintf("Next: run 'foundryctl forge -f %s' to generate deployment files", result.OutputPath))

	return nil
}

func formatDeployment(d v1alpha1.TypeDeployment) string {
	if d.Platform != "" && d.Mode != "" {
		return fmt.Sprintf("%s/%s/%s", d.Platform, d.Mode, d.Flavor)
	}
	if d.Platform != "" {
		return fmt.Sprintf("%s/%s", d.Platform, d.Flavor)
	}
	return fmt.Sprintf("%s/%s", d.Mode, d.Flavor)
}
