package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
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
	cueFileParsed     cue.Value
)

const (
	module     = "github.com/signoz/foundry"
	schemaPath = "./internal/schema/casting.cue"
)

// func loadSchema loads and compiles the CUE schema from the specified path.
func loadSchema(ctx *cue.Context, logger *slog.Logger) (cue.Value, error) {
	cfg := &load.Config{
		Dir:    "../../", // Moved to the root of the repository in latest changes, adjusting
		Module: module,
	}
	logger.Debug("Loading and compiling schema", slog.String("schema.path", schemaPath))

	instance := load.Instances([]string{schemaPath}, cfg)
	if len(instance) == 0 {
		return cue.Value{}, fmt.Errorf("no instances loaded for schema path %s", schemaPath)
	}

	inst := instance[0]
	if inst.Err != nil {
		return cue.Value{}, fmt.Errorf("schema loading error:\n%s", errors.Details(inst.Err, nil))
	}

	value := ctx.BuildInstance(inst)
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("schema compilation error:\n%s", errors.Details(value.Err(), nil))
	}

	return value, nil
}

// func RunGauge is the main function for the gauge command.
func runGauge(cmd *cobra.Command, _ []string) error {
	logger := instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "gauge"))
	ctx := cmd.Context()
	logger.DebugContext(ctx, "Starting Gauge command, using:", slog.String("cfg.file", cfg.File))
	config, err := validateConfig(cfg.File, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to validate config", slog.String("cfg.file", cfg.File), slog.String("error", err.Error()))
		return err
	}
	requirements, err := getRequirements(config, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get requirements from config", slog.String("cfg.file", cfg.File), slog.String("error", err.Error()))
		return err
	}
	logger.InfoContext(ctx, "Required tools:", slog.String("tools", fmt.Sprintf("%v", requirements)))
	for _, v := range requirements {
		err := checkToolExists(v)
		if err != nil {
			logger.WarnContext(ctx, "Tool not found", slog.String("tool.name", v))
		} else {
			logger.InfoContext(ctx, "Tool is available", slog.String("tool.name", v))
		}
	}
	return nil
}

// func compileDataFile compiles the configuration file based on its extension.
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
		return cue.Value{}, fmt.Errorf("unsupported file format: %s (supported: .yaml, .yml, .json)", ext)
	}

	if expr.Err() != nil {
		return cue.Value{}, fmt.Errorf("config parsing error:\n%s", errors.Details(expr.Err(), nil))
	}

	return expr, err
}

// func validateConfig validates the configuration file against the schema.
func validateConfig(filename string, logger *slog.Logger) (cue.Value, error) {
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return cueFileParsed, errorFilenotFound
	}
	logger.Debug("Read configuration file", slog.String("file.path", filename))

	ctx := cuecontext.New()

	schema, err := loadSchema(ctx, logger)
	if err != nil {
		return cueFileParsed, fmt.Errorf("schema compilation error:\n%s", errors.Details(err, nil))
	}
	schemaString, _ := schema.String()
	logger.Debug("Schema loaded:", slog.String("schema", schemaString))

	// Compile data based on file extension
	data, err := compileDataFile(ctx, filename, configFile)
	if err != nil {
		return cueFileParsed, err
	}

	// Lookup #Config definition
	configSchema := schema.LookupPath(cue.ParsePath("#Config"))
	if configSchema.Err() != nil {
		return cueFileParsed, fmt.Errorf("#Config not found in schema:\n%s", errors.Details(configSchema.Err(), nil))
	}

	// Unify and validate
	unified := configSchema.Unify(data)
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		// Use errors.Details for much better error messages``
		logger.Error("Validation failed")
		return cueFileParsed, fmt.Errorf("validation failed: %s", errors.Details(err, nil))
	}

	cueFileParsed = unified

	logger.Info("✓ Valid Configuration")
	return cueFileParsed, nil
}

// func getRequirements extracts the list of required tools from the CUE configuration.
func getRequirements(cueFile cue.Value, logger *slog.Logger) ([]string, error) {
	var requirements []string
	reqList := cueFile.LookupPath(cue.ParsePath("requirements"))
	logger.Debug("Looking up requirements")
	if reqList.Err() != nil {
		return requirements, fmt.Errorf("failed to lookup requirements:\n%s", errors.Details(reqList.Err(), nil))
	}
	iter, err := reqList.List()
	if err != nil {
		return requirements, fmt.Errorf("failed to iterate requirements:\n%s", errors.Details(err, nil))
	}
	for iter.Next() {
		req := iter.Value()
		reqStr, err := req.String()
		if err != nil {
			return requirements, fmt.Errorf("failed to convert requirement to string:\n%s", errors.Details(err, nil))
		}
		requirements = append(requirements, reqStr)
	}
	return requirements, nil
}

// func checkToolExists validates the tool is installed on the system.
func checkToolExists(toolName string) error {
	_, err := exec.LookPath(toolName)
	return err
}

// func registerGaugeCmd registers the gauge command with the root command.
func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE:  runGauge,
	}

	rootCmd.AddCommand(gaugeCmd)
}
