package main

import "github.com/spf13/cobra"

var (
	// Stores common configuration across all commands.
	commonCfg commonConfig

	// Stores pours configuration.

	poursCfg poursConfig

	// Stores cast configuration.
	castCfg castConfig
)

type commonConfig struct {
	File  string
	Debug bool
}

func (c *commonConfig) RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&c.File, "file", "f", "casting.yaml", "Path to the casting configuration file.")
	cmd.PersistentFlags().BoolVarP(&c.Debug, "debug", "d", false, "Enable debug mode.")
}

type poursConfig struct {
	Path string
}

func (c *poursConfig) RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&c.Path, "pours", "p", "./pours", "Directory for pours containing the deployment and configuration files")
}

type castConfig struct {
	NoGauge bool
	NoForge bool
}

func (c *castConfig) RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&c.NoGauge, "no-gauge", false, "Do not run gauge before forge and cast.")
	cmd.PersistentFlags().BoolVar(&c.NoForge, "no-forge", false, "Do not run forge before cast.")
}
