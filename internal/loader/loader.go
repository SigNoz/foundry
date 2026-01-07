// Package loader package provides functionality to load, validate, and unify
package loader

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/yaml"

	"github.com/signoz/foundry/internal/schema"
)

var (
	errorFilenotFound = errors.New("File not found")
)

// LoadedConfig holds the cue values, used for generation of configs.
type LoadedConfig struct {
	Unified           cue.Value       // User config merged with defaults
	Platform          string          // Deployment platform (docker, linux, etc.)
	SchemaVersion     string          // Schema version from config
	EnabledComponents map[string]bool // Map of component name -> enabled status
}

// LoadSchema loads the CUE schema from the specified path.
// TODO: Could be used to load different schemas for different components (clickhouse, keeper, etc).
func LoadSchema(ctx *cue.Context, filename string) (cue.Value, error) {
	// Build the overlay
	overlay := map[string]load.Source{}

	// Walk through the embedded schema files and add them to the overlay
	err := fs.WalkDir(schema.Content, ".", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		data, _ := fs.ReadFile(schema.Content, path)
		// Prefix with "/" to make it an absolute virtual path
		overlay["/"+path] = load.FromBytes(data)
		return nil
	})

	if err != nil {
		return cue.Value{}, fmt.Errorf("failed to read embedded schema files: %w", err)
	}

	// Configure the loader to use the schema
	cfg := &load.Config{
		Dir:     "/",
		Overlay: overlay,
	}

	// NOTE: This could be used to dynamically load all cue files that have X filename
	// Load the schema files from the overlay
	insts := load.Instances([]string{filename}, cfg)
	for _, inst := range insts {
		if inst.Err != nil {
			return cue.Value{}, fmt.Errorf("schema loading error:\n%s", errors.Details(inst.Err, nil))
		}
	}

	// Return the first instance (there should only be one)
	return ctx.BuildInstance(insts[0]), nil
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

// // ValidateConfig validates the user configuration file against the schema.
// func ValidateConfig(filename string) error {
// 	unified, err := Unify(filename)
// 	if err != nil {
// 		return err
// 	}

// 	if err := unified.Validate(cue.Concrete(true)); err != nil {
// 		return fmt.Errorf("validation failed: %s", errors.Details(err, nil))
// 	}
// 	return nil
// }

// Unify loads and merges the user configuration file with the schema defaults.
func Unify(ctx *cue.Context, filename string) (cue.Value, error) {
	// Read file
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return cue.Value{}, errorFilenotFound
	}


	schema, err := LoadSchema(ctx, "casting.cue")
	if err != nil {
		return cue.Value{}, fmt.Errorf("schema compilation error:\n%s", errors.Details(err, nil))
	}

	data, err := compileDataFile(ctx, filename, configFile)
	if err != nil {
		return cue.Value{}, err
	}

	// Get #Config schema definition
	configSchema := schema.LookupPath(cue.ParsePath("#Config"))
	if configSchema.Err() != nil {
		return cue.Value{}, fmt.Errorf("#Config not found in schema:\n%s", errors.Details(configSchema.Err(), nil))
	}

	return configSchema.Unify(data), nil
}

// LoadConfig loads and validates the casting configuration, returning the parsed config
// with defaults applied. This is used by the forge command to generate deployment files.
func LoadConfig(cuectx *cue.Context, filename string) (*LoadedConfig, error) {

	unified, err := Unify(cuectx, filename)
	if err != nil {
		return &LoadedConfig{}, err
	}

	if err := unified.Validate(cue.Concrete(true)); err != nil {
		return &LoadedConfig{}, errors.New("validation failed:" + err.Error())
	}

	// Extract metadata
	platform, _ := unified.LookupPath(cue.ParsePath("platform")).String()
	schemaVersion, _ := unified.LookupPath(cue.ParsePath("schemaVersion")).String()

	// Build enabled components map by decoding components to a map
	enabled := make(map[string]bool)
	components := unified.LookupPath(cue.ParsePath("components"))

	var componentsMap map[string]map[string]interface{}
	if err := components.Decode(&componentsMap); err == nil {
		for name, compData := range componentsMap {
			if enabledVal, ok := compData["enabled"]; ok {
				if isEnabled, ok := enabledVal.(bool); ok && isEnabled {
					enabled[name] = true
				}
			}
		}
	}

	return &LoadedConfig{
		Unified:           unified,
		Platform:          platform,
		SchemaVersion:     schemaVersion,
		EnabledComponents: enabled,
	}, nil
}
