// Package docker implements Docker platform-specific file generation.
package docker

import (
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
	logger := instrumentation.NewLogger(false).With(slog.String("platform.generator", "docker"))
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
	logger.Debug("Replica Configuration:", slog.Any("replicas", replicaConfig))

	// Read the Docker compose schema
	deployment, err := loader.LoadSchema(ctx, "castings/docker/docker.cue")
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("schema compilation error: %w", err)
	}

	// Generate a map of versions for enabled components
	versionKeyMap := map[string]string{
		"signoz":              "SIGNOZ_VERSION",
		"clickhouse":          "CLICKHOUSE_VERSION",
		"signozOtelCollector": "OTELCOL_VERSION",
	}

	replicaKeyMap := map[string]int{
		"clickhouse":    replicaConfig["clickhouse"],
		"otelcollector": replicaConfig["signozOtelCollector"],
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

	// Generate and merge cluster information for ClickHouse
	if replicaKeyMap["clickhouse"] > 0 {
		clickhouseReplicas := generateClusterNodes("clickhouse", replicaKeyMap["clickhouse"])
		clickhousekeeperReplicas := generateClusterNodes("clickhouse-keeper", replicaKeyMap["clickhouse"])
		zooKeeperReplicas := generateClusterNodes("zookeeper", replicaKeyMap["clickhouse"])

		// Merge ClickHouse replicas
		shardConfig := []map[string]any{
			{"replica": clickhouseReplicas},
		}

		// Merge ClickHouse Keeper replicas
		keeperConfig := map[string]any{
			"server": clickhousekeeperReplicas,
		}

		zookeeperConfig := map[string]any{
			"node": zooKeeperReplicas,
		}

		// Clickhouse Cluster
		config = mergeValues(config, "components.clickhouse.config.serverConfig.remote_servers.cluster.shard", ctx.Encode(shardConfig))
		// Zookeeper Nodes
		config = mergeValues(config, "components.clickhouse.config.serverConfig.zookeeper", ctx.Encode(zookeeperConfig))
		// Kepeer Nodes
		config = mergeValues(config, "components.clickhouse.config.serverConfig.keeper_server.raft_configuration", ctx.Encode(keeperConfig))

		logger.Debug("Merged ClickHouse cluster configuration", slog.Int("replicas", replicaKeyMap["clickhouse"]))
	}

	// Lookup the compose section
	deployment = deployment.LookupPath(cue.ParsePath("compose"))

	// Check if deployment is empty
	if deployment.Err() != nil {
		return cue.Value{}, nil, fmt.Errorf("failed to lookup compose in deployment: %w", deployment.Err())
	}

	// Return the contents as YAML
	var data map[string]any
	yamlBytes, err := cueyaml.Encode(deployment)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("failed to encode deployment to YAML: %w", err)
	}
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
// returns the retrieved CUE value or an error if the lookup fails.
func getValue(c cue.Value, path string) (cue.Value, error) {
	value := c.LookupPath(cue.ParsePath(path))
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("failed to lookup path %s: %w", path, value.Err())
	}
	return value, nil
}

// mergeValues merges a value into the CUE configuration at the specified path.
// Returns the updated CUE value.
func mergeValues(c cue.Value, path string, value cue.Value) cue.Value {
	return c.FillPath(cue.ParsePath(path), value)
}

// This is a recursive function that traverses the entire map.
// Removes keys from the docker compose yaml.
// Specifically removes "params" and "replicas" keys.
func removeParams(data map[string]any) {
	delete(data, "params")
	delete(data, "replicas")
	for _, v := range data {
		if m, ok := v.(map[string]any); ok {
			removeParams(m)
		}
	}
}

// generateClusterNodes generates cluster node configuration for the specified component type.
// Supports: "clickhouse", "clickhouse-keeper""
// Returns a slice of maps with host/port pairs or full server configuration with id, host, peerPort, electionPort.
func generateClusterNodes(componentType string, replicas int) []map[string]any {
	var nodeList []map[string]any

	switch componentType {
	case "clickhouse":
		// ClickHouse replica configuration (for remote_servers)
		for replica := 1; replica <= replicas; replica++ {
			nodeList = append(nodeList, map[string]any{
				"host": fmt.Sprintf("clickhouse-%d", replica),
				"port": 9000,
			})
		}

	case "clickhouse-keeper":
		// ClickHouse Keeper cluster configuration
		for replica := 1; replica <= replicas; replica++ {
			nodeList = append(nodeList, map[string]any{
				"id":       replica,
				"hostname": fmt.Sprintf("clickhouse-%d", replica),
				"port":     9234,
			})
		}

	case "zookeeper":
		// ClickHouse replica configuration (for remote_servers)
		for replica := 1; replica <= replicas; replica++ {
			nodeList = append(nodeList, map[string]any{
				"host": fmt.Sprintf("clickhouse-%d", replica),
				"port": 9181,
			})
		}

	}

	return nodeList
}
