package linux

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"

	"github.com/signoz/foundry/internal/loader"
)

type PlatformGenerator struct{}

// Helper function to generate a service file.
func (g *PlatformGenerator) generateServiceFile(deployment cue.Value, servicePath, fileName string) ([]byte, error) {
	service := deployment.LookupPath(cue.ParsePath(servicePath))
	if service.Err() != nil {
		return nil, service.Err()
	}

	output := service.LookupPath(cue.ParsePath("output"))
	if output.Err() != nil {
		return nil, output.Err()
	}

	out, err := output.String()
	if err != nil {
		return nil, errors.New("Failed to unmarshal the service: " + fileName + err.Error())
	}

	return []byte(out), nil
}

// Generate linux platform-specific files.
func (g *PlatformGenerator) Generate(
	ctx *cue.Context,
	config cue.Value,
	enabledComponents map[string]bool,
) (cue.Value, map[string][]byte, error) {

	deployment, err := loader.LoadSchema(ctx, "castings/linux/linux.cue")
	if err != nil {
		return cue.Value{}, nil, errors.New("schema compilation error: " + err.Error())
	}

	files := make(map[string][]byte)

	// Apply Linux-specific configuration overrides using CUE
	linuxOverrides := deployment.LookupPath(cue.ParsePath("#LinuxOverrides"))
	if linuxOverrides.Err() == nil {
		// First, fill in the _#inputs with concrete values
		inputsValue := ctx.CompileString(`{
			_#inputs: {
				clickhouseHost: "127.0.0.1"
				clickhousePort: "9000"
			}
		}`)
		
		// Unify the definition with the input values
		filledOverrides := linuxOverrides.Unify(inputsValue).LookupPath(cue.ParsePath("out"))
		if filledOverrides.Err() != nil {
			return cue.Value{}, nil, errors.New(
				"failed to fill inputs: " + filledOverrides.Err().Error(),
			)
		}
		
		// Decode overrides to a map to work with concrete values only
		var overridesMap map[string]interface{}
		if err := filledOverrides.Decode(&overridesMap); err != nil {
			return cue.Value{}, nil, errors.New(
				"failed to Decode inputs: " + filledOverrides.Err().Error(),
			)
		}
			overridesValue := ctx.Encode(overridesMap)
			
			componentsPath := cue.ParsePath("components")
			currentComponents := config.LookupPath(componentsPath)
			
			// Unify concrete values
			mergedComponents := currentComponents.Unify(overridesValue)
			
			config = config.FillPath(componentsPath, mergedComponents)
			if config.Err() != nil {
				return cue.Value{}, nil, errors.New(
					"failed to apply linux overrides: " + config.Err().Error(),
				)
			}
	}

	// Generate systemd service files for enabled components
	for componentName, isEnabled := range enabledComponents {
		if !isEnabled {
			continue
		}

		switch componentName {
		case "signoz":
			content, err := g.generateServiceFile(deployment, "#SignozService", "signoz")
			if err != nil {
				return cue.Value{}, nil, err
			}
			files["signoz.service"] = content

		case "signozOtelCollector":
			content, err := g.generateServiceFile(deployment, "#SignozOtelCollectorService", "signoz-otel-collector")
			if err != nil {
				return cue.Value{}, nil, err
			}
			files["signoz-otel-collector.service"] = content

		case "postgres":
			content, err := g.generateServiceFile(deployment, "#PostgresService", "postgres")
			if err != nil {
				return cue.Value{}, nil, err
			}
			files["postgres.service"] = content

		case "zookeeper":
			content, err := g.generateServiceFile(deployment, "#ZookeeperService", "zookeeper")
			if err != nil {
				return cue.Value{}, nil, err
			}
			files["zookeeper.service"] = content
		}
	}

	return config, files, nil
}
