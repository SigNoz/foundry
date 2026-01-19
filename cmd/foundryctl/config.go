package main

import "github.com/spf13/cobra"

var (
	// Stores input configuration.
	cfg config
	// Stores output configuration.
	out output
	// Stores pours configuration.
	pours pour
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

type pour struct {
	Path string
}

func (o *output) RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&o.Path, "output", "o", "./pours", "Output Directory for pours containing the deployment and configuration files")
}

func (p *pour) RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&p.Path, "pours", "p", "", "Input Directory for already existing pours containing the deployment and configuration files")

}
