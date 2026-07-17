package main

import (
	"os"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/version"
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

	err := rootCmd.Execute()

	if rootNotifier != nil {
		rootNotifier.Finish(version.Info.Version(), os.Stderr)
	}

	closeRoot()
	if err != nil {
		os.Exit(foundryerrors.ExitCode(err))
	}
}
