package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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
	registerUncastCmd(rootCmd)
	registerGenCmd(rootCmd)
	registerCatalogCmd(rootCmd)
	registerVersionCmd(rootCmd)

	// Foundry survives the interrupt to keep reading the tool's streams: dying
	// here SIGPIPEs the tool mid-write. Exec'd tools get the kernel's copy
	// directly; the cancelled context is the relay only for in-process SDK work.
	// A second signal kills foundry by the OS default.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		stop()
	}()

	err := rootCmd.ExecuteContext(ctx)

	if rootNotifier != nil {
		rootNotifier.Finish(version.Info.Version(), os.Stderr)
	}

	closeRoot()
	if err != nil {
		os.Exit(foundryerrors.ExitCode(err))
	}
}
