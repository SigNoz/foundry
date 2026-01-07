// Package docker implements Docker platform-specific file generation.
package docker

import (
	"errors"
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	cueyaml "cuelang.org/go/encoding/yaml"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/loader"
	stdyaml "gopkg.in/yaml.v3"
)

// PlatformGenerator is responsible for generating Docker platform files.
type PlatformGenerator struct{}

// Generate docker platform-specific files.
func (g *PlatformGenerator) Generate(
	ctx *cue.Context,
	config cue.Value,
	enabledComponents map[string]bool,
) (cue.Value, map[string][]byte, error) {
	logger := instrumentation.NewLogger(false).With("platform.generator", "docker")
	logger.Debug("Starting Docker platform generation")

	componentVersions := make(map[string]string)
	replicaConfig := make(map[string]int)
	for k := range enabledComponents {
		versionValue, err := getValue(config, fmt.Sprintf("components.%s.version", k))
		if err != nil {
			return cue.Value{}, nil, fmt.Errorf("failed to get version for component %s: %w", k, err)
		}
		versionStr, err := versionValue.String()
		if err != nil {
			return cue.Value{}, nil, fmt.Errorf("failed to convert version to string for component %s: %w", k, err)
		}
		replicaValue, err := getValue(config, fmt.Sprintf("components.%s.replicas", k))
		if err != nil {
			return cue.Value{}, nil, fmt.Errorf("failed to get replicas for component %s: %w", k, err)
		}
		replicaInt, err := replicaValue.Int64()
		if err != nil {
			return cue.Value{}, nil, fmt.Errorf("failed to convert replicas to int for component %s: %w", k, err)
		}

		componentVersions[k] = versionStr
		replicaConfig[k] = int(replicaInt)
	}
	logger.Debug("Component Versions:", slog.Any("versions", componentVersions))
	logger.Debug("Replica Configuration:", slog.Any("versions", componentVersions))

	// Read the Docker compose schema
	deployment, err := loader.LoadSchema(ctx, "castings/docker/docker.cue")
	if err != nil {
		return cue.Value{}, nil, errors.New("schema compilation error: " + err.Error())
	}

	// Generate a map of versions for enabled components
	versionKeyMap := map[string]string{
		"signoz":              "SIGNOZ_VERSION",
		"zookeeper":           "ZOOKEEPER_VERSION",
		"clickhouse":          "CLICKHOUSE_VERSION",
		"signozOtelCollector": "OTELCOL_VERSION",
	}

	replicaKeyMap := map[string]int{
		"zookeeper":  replicaConfig["zookeeper"],
		"clickhouse": replicaConfig["clickhouse"],
	}

	// Iterate over the component versions to merge with deployment lookups.
	for component, version := range componentVersions {
		key, ok := versionKeyMap[component]
		if !ok {
			logger.Warn("No version key mapping found for component", slog.String("component", component))
			continue
		}
		versionValue := ctx.Encode(version)
		deployment = mergeValues(deployment, fmt.Sprintf("compose.params.%s", key), versionValue)
	}

	// Iterate over the replica configurations to merge with deployment lookups.
	for component, replicas := range replicaKeyMap {
		replicaValue := ctx.Encode(replicas)
		deployment = mergeValues(deployment, fmt.Sprintf("compose.replicas.%s", component), replicaValue)
	}

	// Lookup the compose section
	deployment = deployment.LookupPath(cue.ParsePath("compose"))

	// Return the contents as YAML
	var data map[string]any
	yamlBytes, _ := cueyaml.Encode(deployment)
	if err = stdyaml.Unmarshal(yamlBytes, &data); err != nil {
		return cue.Value{}, nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	removeParams(data)
	if yamlBytes, err = stdyaml.Marshal(data); err != nil {
		return cue.Value{}, nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return config, map[string][]byte{"docker-compose.yml": yamlBytes}, nil
}

// getValue retrieves a value from the CUE configuration based on the provided path.
func getValue(c cue.Value, path string) (cue.Value, error) {
	value := c.LookupPath(cue.ParsePath(path))
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("failed to lookup path %s: %w", path, value.Err())
	}
	return value, nil
}

// NOTE: To be used for other configurations
// mergeValues merges a value into the CUE configuration at the specified path.
func mergeValues(c cue.Value, path string, value cue.Value) cue.Value {
	return c.FillPath(cue.ParsePath(path), value)
}

// Removes keys from the docker compose yaml.
func removeParams(data map[string]any) {
	delete(data, "params")
	delete(data, "replicas")
	for _, v := range data {
		if m, ok := v.(map[string]any); ok {
			removeParams(m)
		}
	}
}
