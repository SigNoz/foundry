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
}

// newTracker creates a Ledger based on config.
// Returns a no-op ledger when --no-ledger is passed.
func newTracker() ledger.Ledger {
	config := ledger.NewConfig()
	if commonCfg.NoLedger {
		config.Enabled = false
	}

	switch config.Provider() {
	case "segment":
		return segmentledger.New(config)
	default:
		return noopledger.New()
	}
}
