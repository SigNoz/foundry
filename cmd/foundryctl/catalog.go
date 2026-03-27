package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
)

func registerCatalogCmd(rootCmd *cobra.Command) {
	catalogCmd := &cobra.Command{
		Use:   "catalog",
		Short: "Show available castings",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := instrumentation.NewLogger(commonCfg.Debug)

			return runList(logger)
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

func runList(logger *slog.Logger) error {
	f, err := foundry.New(logger)
	if err != nil {
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

	if catalogCfg.Output == "json" {
		data, err := json.MarshalIndent(map[string]any{"Castings": entries}, "", "  ")
		if err != nil {
			return err
		}
		if catalogCfg.OutPath != "" {
			return os.WriteFile(catalogCfg.OutPath, data, 0644)
		}
		_, err = os.Stdout.Write(data)
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "MODE\tFLAVOR\tPLATFORM\tEXAMPLE")
	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Mode, e.Flavor, e.Platform, e.Example)
	}

	return w.Flush()
}
