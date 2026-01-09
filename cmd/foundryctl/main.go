package main

import (
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
	}

	// Register configuration.
	cfg.RegisterFlags(rootCmd)

	// Initialize instrumentation for the cmd/ package.
	logger := instrumentation.NewLogger(false)

	// Register commands.
	registerGaugeCmd(rootCmd)
	registerForgeCmd(rootCmd)
	registerCastCmd(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.ErrorContext(rootCmd.Context(), "failed to execute command", foundryerrors.LogAttr(err))
		os.Exit(1)
	}
}
