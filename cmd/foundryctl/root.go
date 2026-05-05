package main

import (
	"log/slog"
	"runtime"

	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/ledger"
	"github.com/signoz/foundry/internal/ledger/noopledger"
	"github.com/signoz/foundry/internal/ledger/segmentledger"
	"github.com/spf13/cobra"
)

var (
	rootLogger  *slog.Logger
	rootTracker ledger.Ledger
)

// newRoot is wired as rootCmd.PersistentPreRunE so it fires after persistent
// flags are parsed and before any command's RunE runs.
func newRoot(_ *cobra.Command, _ []string) error {
	rootLogger = instrumentation.NewLogger(commonCfg.Debug)

	config := ledger.NewConfig()
	if commonCfg.NoLedger {
		config.Enabled = false
	}

	switch config.Provider() {
	case "segment":
		rootTracker = segmentledger.New(config)
	default:
		rootTracker = noopledger.New()
	}

	return nil
}

func closeRoot() {
	if rootTracker != nil {
		_ = rootTracker.Close()
	}
}

func recoverRunE(
	event domain.Event,
	runE func(cmd *cobra.Command, args []string) (domain.Properties, error),
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		props := domain.NewProperties()

		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)

				errp := foundryerrors.Newf(foundryerrors.TypeFatal, "%v", r).WithStacktrace(string(buf[:n]))
				err = errp
			}

			if err != nil {
				rootLogger.ErrorContext(ctx, event.String()+" failed", foundryerrors.LogAttr(err))
				rootTracker.Track(ctx, event.Failed(), props.WithError(err))
				return
			}

			rootTracker.Track(ctx, event.Succeeded(), props.WithSuccess())
		}()

		props, err = runE(cmd, args)
		return err
	}
}
