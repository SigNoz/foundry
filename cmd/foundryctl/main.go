package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "foundryctl",
		SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRunE: newRoot,
	}

	// Register configuration.
	commonCfg.RegisterFlags(rootCmd)
	poursCfg.RegisterFlags(rootCmd)

	// Register commands.
	registerGaugeCmd(rootCmd)
	registerForgeCmd(rootCmd)
	registerCastCmd(rootCmd)
	registerGenCmd(rootCmd)
	registerCatalogCmd(rootCmd)
	registerVersionCmd(rootCmd)

	defer closeRoot()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
