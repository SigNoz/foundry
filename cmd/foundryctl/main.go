package main

import (
	"context"
	"os"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "foundryctl",
		SilenceUsage:  true,
		SilenceErrors: true,
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
	closeRoot()
	if err != nil {
		logger := rootLogger
		if logger == nil {
			logger = instrumentation.NewLogger(false)
		}
		logger.ErrorContext(context.Background(), "failed to run foundryctl", foundryerrors.LogAttr(err))
		os.Exit(1)
	}
}
