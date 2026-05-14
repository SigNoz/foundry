package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/foundry"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
	"github.com/swaggest/jsonschema-go"
)

const moduleAPIPrefix = "github.com/signoz/foundry/api/v1alpha1/"

var schemaTargets = []any{
	installation.Casting{},
}

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
			return runGenSchemas(cmd.Context())
		},
	}

	rootCmd.AddCommand(genSchemasCmd)
}

func runGenExamples(ctx context.Context, logger *slog.Logger) error {
	foundry, err := foundry.New(logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create foundry, please report this issues to developers at https://github.com/signoz/foundry/issues", foundryerrors.LogAttr(err))
		return err
	}

	for deployment := range foundry.Registry.CastingItems() {
		logger.InfoContext(ctx, "generating example files for deployment", slog.Any("deployment", deployment))

		config := installation.Example()
		config.Spec.Deployment = deployment

		rootPath := filepath.Join("docs", "examples/", deployment.Platform.String(), deployment.Mode.String(), deployment.Flavor.String())
		err = os.MkdirAll(rootPath, 0755)
		if err != nil {
			return err
		}

		err = os.WriteFile(filepath.Join(rootPath, "casting.yaml"), domain.MustMarshalYAML(config), 0644)
		if err != nil {
			return err
		}

		_, err = runForge(ctx, logger, filepath.Join(rootPath, "casting.yaml"), filepath.Join(rootPath, "pours"))
		if err != nil {
			logger.ErrorContext(ctx, "failed to forge casting", slog.Any("deployment", deployment), foundryerrors.LogAttr(err))
			continue
		}
	}

	return nil
}

func runGenSchemas(_ context.Context) error {
	reflector := jsonschema.Reflector{}
	var oneOf []jsonschema.SchemaOrBool

	for _, t := range schemaTargets {
		schema, err := reflector.Reflect(t)
		if err != nil {
			return fmt.Errorf("reflect %T: %w", t, err)
		}
		contents, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal %T: %w", t, err)
		}

		kindDir := strings.TrimPrefix(reflect.TypeOf(t).PkgPath(), moduleAPIPrefix)
		if err := os.WriteFile(filepath.Join("api", "v1alpha1", kindDir, "casting.schema.json"), contents, 0644); err != nil {
			return err
		}
		ref := (&jsonschema.Schema{}).WithRef(filepath.Join(kindDir, "casting.schema.json"))
		oneOf = append(oneOf, ref.ToSchemaOrBool())
	}

	root := (&jsonschema.Schema{}).
		WithSchema("http://json-schema.org/draft-07/schema#").
		WithTitle("Foundry Casting").
		WithOneOf(oneOf...)

	rootContents, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join("api", "v1alpha1", "casting.schema.json"), rootContents, 0644)
}
