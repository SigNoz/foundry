package main

import (
	"log/slog"
	"os"
	"runtime"

	"github.com/signoz/foundry/api/v1alpha1"
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

// recoverRunE tracks one event per casting document the command touched, all
// carrying the same run id so a multi-document invocation still counts as one
// run.
func recoverRunE(
	event domain.Event,
	runE func(cmd *cobra.Command, args []string) ([]domain.Properties, error),
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		runID := domain.NewRunID()
		props := []domain.Properties{}

		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)

				errp := foundryerrors.Newf(foundryerrors.TypeFatal, "%v", r).WithStacktrace(string(buf[:n]))
				err = errp
			}

			// A run that failed before it read any document still reports.
			if len(props) == 0 {
				props = []domain.Properties{domain.NewProperties()}
			}

			if err != nil {
				rootLogger.ErrorContext(ctx, event.String()+" failed", foundryerrors.LogAttr(err))
				for _, p := range props {
					rootTracker.Track(ctx, event.Failed(), p.WithRunID(runID).WithError(err))
				}

				if commonCfg.Format == "json" {
					_ = writer.WriteOutput(os.Stdout, foundryerrors.EnvelopeOf(err))
					cmd.SilenceErrors = true
				}
				return
			}

			for _, p := range props {
				rootTracker.Track(ctx, event.Succeeded(), p.WithRunID(runID).WithSuccess())
			}
		}()

		props, err = runE(cmd, args)
		return err
	}
}

// trackableProperties is one properties bag per document, so a run reports
// every casting it touched rather than only the first.
func trackableProperties(machineries []v1alpha1.Machinery) []domain.Properties {
	props := make([]domain.Properties, 0, len(machineries))
	for _, machinery := range machineries {
		props = append(props, machinery.TrackableProperties())
	}

	return props
}
