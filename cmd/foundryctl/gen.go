package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	installationcasting "github.com/signoz/foundry/internal/casting/installation"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
	"github.com/swaggest/jsonschema-go"
)

const moduleAPIPrefix = "github.com/signoz/foundry/api/v1alpha1/"

type schemaTarget struct {
	kind v1alpha1.Kind
	val  any
}

var schemaTargets = []schemaTarget{
	{v1alpha1.KindInstallation, installation.Casting{}},
	{v1alpha1.KindCollectionAgent, collectionagent.Casting{}},
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
	registry := installationcasting.NewRegistry(logger)

	for deployment := range registry.CastingItems() {
		logger.InfoContext(ctx, "generating example files for deployment", slog.Any("deployment", deployment))

		config := installation.Example()
		config.Spec.Deployment = deployment

		rootPath := filepath.Join("docs", "examples/", deployment.Platform.String(), deployment.Mode.String(), deployment.Flavor.String())
		if err := os.MkdirAll(rootPath, 0755); err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(rootPath, "casting.yaml"), domain.MustMarshalYAML(config), 0644); err != nil {
			return err
		}

		if _, err := runForge(ctx, logger, filepath.Join(rootPath, "casting.yaml"), filepath.Join(rootPath, "pours")); err != nil {
			logger.ErrorContext(ctx, "failed to forge casting", slog.Any("deployment", deployment), foundryerrors.LogAttr(err))
			continue
		}
	}

	return nil
}

func runGenSchemas(_ context.Context) error {
	var oneOf []jsonschema.SchemaOrBool
	kindType := reflect.TypeFor[v1alpha1.Kind]()

	for _, t := range schemaTargets {
		target := t
		reflector := jsonschema.Reflector{}
		// v1alpha1.Kind's Enum() returns all Kinds (the type permits any).
		// For this per-Kind schema, the kind field is always this Casting's
		// Kind, so we narrow the enum at reflection.
		reflector.DefaultOptions = append(reflector.DefaultOptions,
			jsonschema.InterceptSchema(func(params jsonschema.InterceptSchemaParams) (bool, error) {
				if !params.Processed || params.Value.Type() != kindType {
					return false, nil
				}
				params.Schema.Enum = []any{target.kind.String()}
				return false, nil
			}),
		)

		schema, err := reflector.Reflect(target.val)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "reflect %T", target.val)
		}

		contents, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "marshal %T", target.val)
		}

		kindDir := strings.TrimPrefix(reflect.TypeOf(target.val).PkgPath(), moduleAPIPrefix)
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
