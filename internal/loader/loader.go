 package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/yaml"
)

var(
	module = "github.com/signoz/foundry"
	schemaPath = "./internal/schema/casting.cue"
	errorFilenotFound   = errors.New("File not found")
)

// LoadedConfig holds the cue values, used for generation of configs.
type LoadedConfig struct {
	Unified           cue.Value         // User config merged with defaults
	Platform          string            // Deployment platform (docker, linux, etc.)
	SchemaVersion     string            // Schema version from config
	EnabledComponents map[string]bool   // Map of component name -> enabled status
}

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


func ValidateConfig(filename string) error {
	unified, err := Unify(filename)
	if err != nil {
		return err
	}

	if err := unified.Validate(cue.Concrete(true)); err != nil {
		return fmt.Errorf("validation failed: %s", errors.Details(err, nil))
	}
	return nil
}

func Unify(filename string)(cue.Value, error){
	// Read file
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return cue.Value{}, errorFilenotFound
	}

	ctx := cuecontext.New()

	schema, err := loadSchema(ctx)
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
func LoadConfig(filename string) (*LoadedConfig, error) {

	unified, err := Unify(filename)
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