package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/yaml"
	"github.com/SigNoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
)

var (
	errorFilenotFound = errors.New("File not found")
)

const (
	module     = "github.com/signoz/foundry"
	schemaPath = "./internal/schema/casting.cue"
)

func loadSchema(ctx *cue.Context, logger *slog.Logger) (cue.Value, error) {
	cfg := &load.Config{
		Dir:    ".",
		Module: module,
	}
	logger.Debug("Loading and compiling schema", slog.String("schema.path", schemaPath))

	instance := load.Instances([]string{schemaPath}, cfg)
	if len(instance) == 0 {
		return cue.Value{}, instance[0].Err
	}

	return ctx.BuildInstance(instance[0]), nil
}

func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger := instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "gauge"))
			ctx := cmd.Context()
			logger.DebugContext(ctx, "Starting Gauge command, using:", slog.String("cfg.file", cfg.File))
			err := validateConfig(cfg.File, logger)
			if err != nil {
				logger.ErrorContext(ctx, "failed to validate config", slog.String("cfg.file", cfg.File), slog.String("error", err.Error()))
			}

			return nil
		},
	}

	rootCmd.AddCommand(gaugeCmd)
}

func compileDataFile(ctx *cue.Context, filename string, data []byte) (cue.Value, error) {
	ext := filepath.Ext(filename)

	var expr cue.Value
	var err error

	switch ext {
	case ".yaml", ".yml":
		yamlExpr, err := yaml.Extract(filename, data)
		if err != nil {
			return cue.Value{}, fmt.Errorf("failed to parse YAML: %w", err)
		}
		expr = ctx.BuildFile(yamlExpr)

	case ".json":
		jsonExpr, err := json.Extract(filename, data)
		if err != nil {
			return cue.Value{}, fmt.Errorf("failed to parse JSON: %w", err)
		}
		expr = ctx.BuildExpr(jsonExpr)

	default:
		return cue.Value{}, fmt.Errorf("unsupported file format: %s (supported: .yaml, .yml, .json, .toml)", ext)
	}

	if expr.Err() != nil {
		return cue.Value{}, fmt.Errorf("config parsing error:\n%s", errors.Details(expr.Err(), nil))
	}

	return expr, err
}

func validateConfig(filename string, logger *slog.Logger) error {
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return errorFilenotFound
	}
	logger.Debug("Read configuration file", slog.String("file.path", filename))

	ctx := cuecontext.New()

	schema, err := loadSchema(ctx, logger)
	if err != nil {
		return fmt.Errorf("schema compilation error:\n%s", errors.Details(err, nil))
	}

	// Compile data based on file extension
	data, err := compileDataFile(ctx, filename, configFile)
	if err != nil {
		return err
	}

	// Lookup #Config definition
	configSchema := schema.LookupPath(cue.ParsePath("#Config"))
	if configSchema.Err() != nil {
		return fmt.Errorf("#Config not found in schema:\n%s", errors.Details(configSchema.Err(), nil))
	}

	// Unify and validate
	unified := configSchema.Unify(data)
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		// Use errors.Details for much better error messages``
		logger.Error("Validation failed")
		return fmt.Errorf("validation failed: %s", errors.Details(err, nil))
	}

	logger.Info("✓ Valid Configuration")
	return nil
}
