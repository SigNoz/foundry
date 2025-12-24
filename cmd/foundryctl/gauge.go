package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/yaml"
	"github.com/SigNoz/foundry/internal/instrumentation"
	"github.com/spf13/cobra"
	"cuelang.org/go/cue/load"
)

var (
	errorFilenotFound   = errors.New("File not found")
	logger              = instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "gauge"))
)

var(
	module = "github.com/signoz/foundry"
	schemaPath = "./internal/schema/casting.cue"
)

func loadSchema(ctx *cue.Context)(cue.Value, error){
	cfg := &load.Config{
		Dir: ".",
		Module: module,
	}

	insts := load.Instances([]string{schemaPath}, cfg)
	if len(insts) == 0{
		return cue.Value{}, insts[0].Err
	}

	return ctx.BuildInstance(insts[0]), nil
}


func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			filePath := cfg.File
			err := validateConfig(filePath)
			if err != nil {
				logger.ErrorContext(ctx, "failed to validate config", slog.String("cfg.file", cfg.File), slog.String("error", err.Error()))
			}

			logger.DebugContext(ctx, "starting command", slog.String("cfg.file", cfg.File))
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


func validateConfig(filename string) error {
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return errorFilenotFound
	}

	ctx := cuecontext.New()

	schema, err := loadSchema(ctx)
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
