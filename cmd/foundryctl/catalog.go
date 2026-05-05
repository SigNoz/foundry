package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"sort"

	"github.com/olekukonko/tablewriter"
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/spf13/cobra"
)

func registerCatalogCmd(rootCmd *cobra.Command) {
	catalogCmd := &cobra.Command{
		Use:   "catalog",
		Short: "Show available castings",
		RunE: recoverRunE(domain.EventCatalog, func(cmd *cobra.Command, args []string) (domain.Properties, error) {
			return runCatalog(rootLogger)
		}),
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
	platform, mode, flavor := d.Platform.String(), d.Mode.String(), d.Flavor.String()
	switch {
	case platform != "" && mode != "":
		return platform + "/" + mode + "/" + flavor
	case platform != "":
		return platform + "/" + flavor
	default:
		return mode + "/" + flavor
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

func runCatalog(logger *slog.Logger) (domain.Properties, error) {
	f, err := foundry.New(logger)
	if err != nil {
		return domain.NewProperties(), err
	}

	var entries []castingEntry
	for d := range f.Registry.CastingItems() {
		entries = append(entries, castingEntry{
			Platform: d.Platform.String(),
			Mode:     d.Mode.String(),
			Flavor:   d.Flavor.String(),
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

	props := domain.NewProperties()

	if catalogCfg.Format == "json" {
		data, err := json.MarshalIndent(map[string]any{"Castings": entries}, "", "  ")
		if err != nil {
			return props, err
		}
		if catalogCfg.OutPath != "" {
			err = os.WriteFile(catalogCfg.OutPath, data, 0644)
			return props, err
		}
		_, err = os.Stdout.Write(data)
		return props, err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Mode", "Flavor", "Platform", "Example")
	for _, e := range entries {
		_ = table.Append(e.Mode, e.Flavor, e.Platform, e.Example)
	}

	err = table.Render()
	return props, err
}
