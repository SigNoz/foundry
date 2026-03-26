package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/types"
	"github.com/spf13/cobra"
	"github.com/swaggest/jsonschema-go"
)

func registerGenCmd(rootCmd *cobra.Command) {
	genCmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate example files for all supported deployments.",
	}

	registerGenExamples(genCmd)
	registerGenSchemas(genCmd)

	rootCmd.AddCommand(genCmd)
}

func registerGenExamples(rootCmd *cobra.Command) {
	genExamplesCmd := &cobra.Command{
		Use:   "examples",
		Short: "Generate example files for all supported deployments.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(commonCfg.Debug)

			return runGenExamples(ctx, logger)
		},
	}

	rootCmd.AddCommand(genExamplesCmd)
}

func registerGenSchemas(rootCmd *cobra.Command) {
	genSchemasCmd := &cobra.Command{
		Use:   "schemas",
		Short: "Generate schema files.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := instrumentation.NewLogger(commonCfg.Debug)

			return runGenSchemas(ctx, logger)
		},
	}

	rootCmd.AddCommand(genSchemasCmd)
}

type castingEntry struct {
	Platform string `json:"Platform"`
	Mode     string `json:"Mode"`
	Flavor   string `json:"Flavor"`
	Path     string `json:"Path"`
}

func runGenExamples(ctx context.Context, logger *slog.Logger) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	var entries []castingEntry
	for deployment := range foundry.Registry.CastingItems() {
		logger.InfoContext(ctx, "generating example files for deployment", slog.Any("deployment", deployment))

		config := v1alpha1.ExampleCasting()
		config.Spec.Deployment = deployment

		rootPath := filepath.Join("docs", "examples/", deployment.Platform, deployment.Mode, deployment.Flavor)
		err = os.MkdirAll(rootPath, 0755)
		if err != nil {
			return err
		}

		err = os.WriteFile(filepath.Join(rootPath, "casting.yaml"), types.MustMarshalYAML(config), 0644)
		if err != nil {
			return err
		}

		err = runForge(ctx, logger, filepath.Join(rootPath, "casting.yaml"), filepath.Join(rootPath, "pours"))
		if err != nil {
			logger.ErrorContext(ctx, "failed to forge casting", slog.Any("deployment", deployment), foundryerrors.LogAttr(err))
			continue
		}

		p := filepath.Join(deployment.Platform, deployment.Mode, deployment.Flavor)
		entries = append(entries, castingEntry{
			Platform: valOrDash(deployment.Platform),
			Mode:     valOrDash(deployment.Mode),
			Flavor:   valOrDash(deployment.Flavor),
			Path:     p,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path > entries[j].Path
	})

	castingsJSON, err := json.MarshalIndent(map[string]any{"Castings": entries}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("docs", "examples/", "castings.json"), castingsJSON, 0644); err != nil {
		return err
	}
	logger.InfoContext(ctx, "generated castings.json")

	return nil
}
func valOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func runGenSchemas(context.Context, *slog.Logger) error {
	reflector := jsonschema.Reflector{}

	schema, err := reflector.Reflect(v1alpha1.Casting{})
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(filepath.Join("docs", "schemas", "v1alpha1.yaml"), types.MustMarshalYAML(schema), 0644)
	if err != nil {
		return err
	}

	return nil
}
