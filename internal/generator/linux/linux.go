package linux

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"

	"github.com/signoz/foundry/internal/common"
	"github.com/signoz/foundry/internal/loader"
	"github.com/signoz/foundry/internal/schema"
)

type PlatformGenerator struct{}

// extractComponentAuth extracts auth configuration from a component config.
func extractComponentAuth(config cue.Value, componentPath string) map[string]interface{} {
	authPath := cue.ParsePath(componentPath + ".config.auth")
	authValue := config.LookupPath(authPath)
	if authValue.Err() != nil {
		return nil
	}

	var auth map[string]interface{}
	if err := authValue.Decode(&auth); err != nil {
		return nil
	}
	return auth
}

// applyOverrides applies platform-specific overrides to the config.
func applyOverrides(ctx *cue.Context, config cue.Value, overrides cue.Value, inputs map[string]interface{}) (cue.Value, error) {
	filledOverrides := overrides.FillPath(cue.ParsePath("inputs"), inputs).LookupPath(cue.ParsePath("out"))
	if filledOverrides.Err() != nil {
		return cue.Value{}, errors.New("failed to fill override inputs with provided values: " + filledOverrides.Err().Error())
	}

	var overridesMap map[string]interface{}
	if err := filledOverrides.Decode(&overridesMap); err != nil {
		return cue.Value{}, errors.New("failed to decode override output to map: " + err.Error())
	}

	overridesValue := ctx.Encode(overridesMap)
	componentsPath := cue.ParsePath("components")
	currentComponents := config.LookupPath(componentsPath)
	mergedComponents := currentComponents.Unify(overridesValue)

	if err := mergedComponents.Validate(cue.Concrete(true)); err != nil {
		return cue.Value{}, errors.New("failed to unify current components with override values: " + mergedComponents.Err().Error())
	}

	config = config.FillPath(componentsPath, mergedComponents)
	if config.Err() != nil {
		return cue.Value{}, errors.New("failed to apply merged components back to config at path 'components': " + config.Err().Error())
	}

	return config, nil
}

// generateServiceFile extracts a service file output from a deployment schema.
func (g *PlatformGenerator) generateServiceFile(deployment cue.Value, servicePath, fileName string) ([]byte, error) {
	service := deployment.LookupPath(cue.ParsePath(servicePath))
	if service.Err() != nil {
		return nil, errors.New("failed to lookup service definition at path '" + servicePath + "' for file '" + fileName + "': " + service.Err().Error())
	}

	output := service.LookupPath(cue.ParsePath("output"))
	if output.Err() != nil {
		return nil, errors.New("failed to lookup output field in service definition '" + servicePath + "' for file '" + fileName + "': " + output.Err().Error())
	}

	out, err := output.String()
	if err != nil {
		return nil, errors.New("failed to convert service output to string for file '" + fileName + "': " + err.Error())
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
		return cue.Value{}, nil, errors.New("failed to load linux deployment schema from 'castings/linux/linux.cue': " + err.Error())
	}

	files := make(map[string][]byte)

	// Apply Linux-specific configuration overrides using CUE
	linuxOverrides := deployment.LookupPath(cue.ParsePath("#Overrides"))
	if linuxOverrides.Err() != nil {
		return cue.Value{}, nil, linuxOverrides.Err()

	}
	// Prepare inputs for the override template
	var clickhouseReplicas int
	replicasPath := cue.ParsePath("components.clickhouse.replicas")
	if replicasValue := config.LookupPath(replicasPath); replicasValue.Err() == nil {
		if r, err := replicasValue.Int64(); err == nil {
			clickhouseReplicas = int(r)
		}
	}
	// Prepare inputs for the override template
	var zookeeperReplicas int
	replicasPath = cue.ParsePath("components.zookeeper.replicas")
	if replicasValue := config.LookupPath(replicasPath); replicasValue.Err() == nil {
		if r, err := replicasValue.Int64(); err == nil {
			zookeeperReplicas = int(r)
		}
	}
	postgresAuth := extractComponentAuth(config, "components.postgres")
	inputsValue := map[string]interface{}{
		"clickhouse": map[string]interface{}{
			"host":     "127.0.0.1",
			"port":     "9000",
			"replicas": clickhouseReplicas,
		},
		"postgres": map[string]interface{}{
			"host": "127.0.0.1",
			"port": "5432",
			"auth": postgresAuth,
		},
		"zookeeper": map[string]interface{}{
			"host":     "127.0.0.1",
			"port":     "2181",
			"replicas": zookeeperReplicas,
		},
	}

	config, err = applyOverrides(ctx, config, linuxOverrides, inputsValue)
	if err != nil {
		return cue.Value{}, nil, errors.New("failed to apply linux platform overrides to configuration: " + err.Error())
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

			// Generate signoz environment file from config
			signozConfig := config.LookupPath(cue.ParsePath("components.signoz.config"))
			if signozConfig.Err() == nil {
				files["signoz.env"], err = common.GenerateSigNozEnv(signozConfig)
				if err != nil {
					return cue.Value{}, nil, errors.New("failed to generate signoz environment file from config: " + err.Error())
				}
			}
		case "signozOtelCollector":
			content, err := g.generateServiceFile(deployment, "#SignozOtelCollectorService", "signoz-otel-collector")
			if err != nil {
				return cue.Value{}, nil, err
			}
			files["signoz-otel-collector.service"] = content

			// Generate opamp config
			opampConfig := []byte("server_endpoint: ws://127.0.0.1:4320/v1/opamp\n")
			files["opamp.yaml"] = opampConfig

		case "zookeeper":
			content, err := g.generateServiceFile(deployment, "#ZookeeperService", "zookeeper")
			if err != nil {
				return cue.Value{}, nil, err
			}
			files["zookeeper.service"] = content

		case "clickhouse":
			content, err := g.generateServiceFile(deployment, "#ClickHouseService", "clickhouse")
			if err != nil {
				return cue.Value{}, nil, err
			}
			files["clickhouse.service"] = content
		}
	}

	files["install.sh"], err = schema.Content.ReadFile("castings/linux/install.sh")
	if err != nil {
		return cue.Value{}, nil, err
	}
	return config, files, nil
}
