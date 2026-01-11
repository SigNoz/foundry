package main

import "github.com/spf13/cobra"

var (
	// Stores input configuration.
	cfg config
	// Stores output configuration.
	out output
)

type config struct {
	File  string
	Debug bool
}

func (c *config) RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&c.File, "file", "f", "casting.yaml", "Path to casting.yaml file.")
	cmd.PersistentFlags().BoolVarP(&c.Debug, "debug", "d", false, "Enable debug mode.")
}

type output struct {
	Path string
}

func (o *output) RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&o.Path, "output", "o", "./pours", "Output Directory for pours containing the deployment and configuration files")
}
