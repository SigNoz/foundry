package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	// Stores common configuration across all commands.
	commonCfg commonConfig

	// Stores pours configuration.
	poursCfg poursConfig

	// Stores cast configuration.
	castCfg castConfig

	// Stores catalog configuration.
	catalogCfg catalogConfig

	// Stores mechanic configuration.
	mechanicCfg mechanicConfig
)

type commonConfig struct {
	File      string
	Debug     bool
	Format    string
	NoLedger  bool
	NoUpdater bool
}

func (c *commonConfig) RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&c.File, "file", "f", "casting.yaml", "Path to the casting configuration file.")
	cmd.PersistentFlags().BoolVarP(&c.Debug, "debug", "d", false, "Enable debug mode.")
	cmd.PersistentFlags().StringVar(&c.Format, "format", "json", "Output format for results and errors (json|text).")
	cmd.PersistentFlags().BoolVar(&c.NoLedger, "no-ledger", false, "Disable anonymous usage ledger.")
	cmd.PersistentFlags().BoolVar(&c.NoUpdater, "no-updater", false, "Disable the update notifier.")
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

type catalogConfig struct {
	OutPath string
}

func (c *catalogConfig) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.OutPath, "output", "o", "", "Path to write castings.json")
}

// mechanicConfig holds connection overrides for the mechanic verbs. They take
// precedence over the resolved casting's status addresses and let mechanic run
// against a deployment that was not provisioned by foundry. Each flag defaults
// to its environment variable so secrets need not appear in shell history.
type mechanicConfig struct {
	Signoz        string
	ClickhouseDSN string
	MetastoreDSN  string
}

func (c *mechanicConfig) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Signoz, "signoz", os.Getenv("FOUNDRY_SIGNOZ"), "Override the SigNoz API address, host:port (default $FOUNDRY_SIGNOZ).")
	cmd.Flags().StringVar(&c.ClickhouseDSN, "clickhouse-dsn", os.Getenv("FOUNDRY_CLICKHOUSE_DSN"), "Override the ClickHouse DSN, user:pass@host:port (default $FOUNDRY_CLICKHOUSE_DSN).")
	cmd.Flags().StringVar(&c.MetastoreDSN, "metastore-dsn", os.Getenv("FOUNDRY_METASTORE_DSN"), "Override the metastore DSN (default $FOUNDRY_METASTORE_DSN).")
}
