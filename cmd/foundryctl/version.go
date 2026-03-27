package main

import (
	"fmt"
	"os"

	"github.com/signoz/foundry/internal/version"
	"github.com/spf13/cobra"
)

var versionCfg versionConfig

type versionConfig struct {
	Short bool
}

func (c *versionConfig) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.Short, "short", false, "Print version in a single line")
}

func registerVersionCmd(rootCmd *cobra.Command) {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			if versionCfg.Short {
				_, _ = fmt.Fprintln(os.Stdout, version.Info.Short())
				return
			}
			version.Info.PrettyPrint()
		},
	}

	versionCfg.RegisterFlags(versionCmd)
	rootCmd.AddCommand(versionCmd)
}
