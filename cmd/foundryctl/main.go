package main

import (
	"context"
	"os"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/ledger"
	"github.com/signoz/foundry/internal/ledger/noopledger"
	"github.com/signoz/foundry/internal/ledger/segmentledger"
	"github.com/spf13/cobra"
)

// tracker is the global ledger instance, initialised in PersistentPreRun.
var tracker ledger.Ledger

func main() {
	rootCmd := &cobra.Command{
		Use:           "foundryctl",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			config := ledger.NewConfig()
			if commonCfg.NoLedger || os.Getenv("FOUNDRY_LEDGER_ENABLED") == "false" {
				config.Enabled = false
			}

			switch config.Provider() {
			case "segment":
				tracker = segmentledger.New(instrumentation.NewLogger(commonCfg.Debug), config)
			default:
				tracker = noopledger.New()
			}
		},
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

	logger := instrumentation.NewLogger(false)

	if err := rootCmd.Execute(); err != nil {
		logger.ErrorContext(context.Background(), "failed to run foundryctl", foundryerrors.LogAttr(err))
		os.Exit(1)
	}

	// Flush any pending ledger events.
	if tracker != nil {
		_ = tracker.Close()
	}
}
