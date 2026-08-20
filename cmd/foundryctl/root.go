package main

import (
	"log/slog"
	"os"
	"runtime"

	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/ledger"
	"github.com/signoz/foundry/internal/ledger/noopledger"
	"github.com/signoz/foundry/internal/ledger/segmentledger"
	"github.com/signoz/foundry/internal/updater"
	"github.com/signoz/foundry/internal/writer"
	"github.com/spf13/cobra"
)

var (
	rootLogger   *slog.Logger
	rootTracker  ledger.Ledger
	rootNotifier *updater.Notifier
)

// newRoot is wired as rootCmd.PersistentPreRunE so it fires after persistent
// flags are parsed and before any command's RunE runs.
func newRoot(cmd *cobra.Command, _ []string) error {
	rootLogger = instrumentation.NewLogger(commonCfg.Debug)

	ledgerConfig := ledger.NewConfig()
	if commonCfg.NoLedger {
		ledgerConfig.Enabled = false
	}

	switch ledgerConfig.Provider() {
	case "segment":
		rootTracker = segmentledger.New(ledgerConfig)
	default:
		rootTracker = noopledger.New()
	}

	updaterConfig := updater.NewConfig()
	if commonCfg.NoUpdater {
		updaterConfig.Enabled = false
	}
	rootNotifier = updater.NewNotifier(updaterConfig, rootLogger)
	rootNotifier.Notify(cmd.Context())

	return nil
}

func closeRoot() {
	if rootTracker != nil {
		_ = rootTracker.Close()
	}
}

// reporter records one casting document's outcome, called once per document
// as it completes; err is nil when the document succeeded.
type reporter func(props domain.Properties, err error)

// recoverRunE hands the runner a reporter that tracks every reported outcome
// as one event. An error the runner never reported tracks without document
// properties; a clean return that reported nothing tracks the lone success.
func recoverRunE(
	event domain.Event,
	runE func(cmd *cobra.Command, args []string, report reporter) error,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		reported, failureReported := false, false

		report := func(props domain.Properties, reportErr error) {
			reported = true

			if reportErr != nil {
				failureReported = true
				rootTracker.Track(ctx, event.Failed(), props.WithError(reportErr))
				return
			}

			rootTracker.Track(ctx, event.Succeeded(), props.WithSuccess())
		}

		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)

				errp := foundryerrors.Newf(foundryerrors.TypeFatal, "%v", r).WithStacktrace(string(buf[:n]))
				err = errp
			}

			if err != nil {
				rootLogger.ErrorContext(ctx, event.String()+" failed", foundryerrors.LogAttr(err))
				if !failureReported {
					rootTracker.Track(ctx, event.Failed(), domain.NewProperties().WithError(err))
				}
				if commonCfg.Format == "json" {
					_ = writer.WriteOutput(os.Stdout, foundryerrors.EnvelopeOf(err))
					cmd.SilenceErrors = true
				}
				return
			}

			if !reported {
				rootTracker.Track(ctx, event.Succeeded(), domain.NewProperties().WithSuccess())
			}
		}()

		err = runE(cmd, args, report)
		return err
	}
}
