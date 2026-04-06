package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sort"

	"github.com/olekukonko/tablewriter"
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/ledger"
	"github.com/spf13/cobra"
)

func registerCatalogCmd(rootCmd *cobra.Command) {
	catalogCmd := &cobra.Command{
		Use:   "catalog",
		Short: "Show available castings",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := instrumentation.NewLogger(commonCfg.Debug)
			tracker := newTracker()
			defer func() {
				_ = tracker.Close()
			}()

			return runCatalog(cmd.Context(), logger, tracker)
		},
	}

	catalogCfg.RegisterFlags(catalogCmd)
	rootCmd.AddCommand(catalogCmd)
}

type castingEntry struct {
	Platform string `json:"platform"`
	Mode     string `json:"mode"`
	Flavor   string `json:"flavor"`
	Example  string `json:"example"`
}

func castingExample(d v1alpha1.TypeDeployment) string {
	switch {
	case d.Platform != "" && d.Mode != "":
		return d.Platform + "/" + d.Mode + "/" + d.Flavor
	case d.Platform != "":
		return d.Platform + "/" + d.Flavor
	default:
		return d.Mode + "/" + d.Flavor
	}
}

// catalogGroup returns a sort key that groups entries:
// 0 = mode + flavor (self-hosted: docker, systemd, kubernetes).
// 1 = mode + flavor + platform (cloud infra: ecs).
// 2 = platform + flavor (managed platforms: coolify, render, railway).
func catalogGroup(e castingEntry) int {
	switch {
	case e.Platform != "" && e.Mode != "":
		return 1
	case e.Platform != "":
		return 2
	default:
		return 0
	}
}

func runCatalog(ctx context.Context, logger *slog.Logger, tracker ledger.Ledger) error {
	f, err := foundry.New(logger)
	if err != nil {
		tracker.Track(ctx, ledger.EventCatalog, ledger.WithError(nil, err))
		return err
	}

	var entries []castingEntry
	for d := range f.Registry.CastingItems() {
		entries = append(entries, castingEntry{
			Platform: d.Platform,
			Mode:     d.Mode,
			Flavor:   d.Flavor,
			Example:  castingExample(d),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		gi, gj := catalogGroup(entries[i]), catalogGroup(entries[j])
		if gi != gj {
			return gi < gj
		}
		return entries[i].Example < entries[j].Example
	})

	if catalogCfg.Format == "json" {
		data, err := json.MarshalIndent(map[string]any{"Castings": entries}, "", "  ")
		if err != nil {
			tracker.Track(ctx, ledger.EventCatalog, ledger.WithError(nil, err))
			return err
		}
		if catalogCfg.OutPath != "" {
			err = os.WriteFile(catalogCfg.OutPath, data, 0644)
			if err != nil {
				tracker.Track(ctx, ledger.EventCatalog, ledger.WithError(nil, err))
				return err
			}
			tracker.Track(ctx, ledger.EventCatalog, ledger.WithSuccess(nil))
			return nil
		}
		_, err = os.Stdout.Write(data)
		if err != nil {
			tracker.Track(ctx, ledger.EventCatalog, ledger.WithError(nil, err))
			return err
		}
		tracker.Track(ctx, ledger.EventCatalog, ledger.WithSuccess(nil))
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Mode", "Flavor", "Platform", "Example")
	for _, e := range entries {
		_ = table.Append(e.Mode, e.Flavor, e.Platform, e.Example)
	}

	err = table.Render()
	if err != nil {
		tracker.Track(ctx, ledger.EventCatalog, ledger.WithError(nil, err))
		return err
	}

	tracker.Track(ctx, ledger.EventCatalog, ledger.WithSuccess(nil))
	return nil
}
