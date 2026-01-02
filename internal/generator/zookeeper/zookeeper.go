package zookeeper

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	casting "github.com/signoz/foundry/internal/schema"
)

type Generator struct{}

func (g *Generator) GenerateComponent(config casting.Config) (map[string][]byte, error) {
	files := make(map[string][]byte)

	zookeeperComponent, exists := config.Components["zookeeper"]
	if !exists {
		return nil, errors.New("zookeeper component not found in config")
	}

	configValue, ok := zookeeperComponent["config"]
	if !ok {
		return nil, errors.New("config key not found in zookeeper component")
	}

	configMap, ok := configValue.(map[string]any)
	if !ok {
		return nil, errors.New("zookeeper config is not a valid map structure")
	}

	// Convert config to zoo.cfg format
	zooCfgBytes := MapToZooCfg(configMap)
	files["zoo.cfg"] = zooCfgBytes

	return files, nil
}

// MapToZooCfg converts to Zookeeper zoo.cfg format
func MapToZooCfg(data map[string]any) []byte {
	var buf bytes.Buffer
	
	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	for _, key := range keys {
		value := data[key]
		
		// Handle servers specially
		if key == "servers" {
			writeServers(&buf, value)
			continue
		}
		
		if nestedMap, ok := value.(map[string]any); ok {
			writeNested(&buf, key, nestedMap)
			continue
		}
		
		buf.WriteString(fmt.Sprintf("%s=%v\n", key, value))
	}
	
	return buf.Bytes()
}

// writeServers handles the servers configuration
func writeServers(buf *bytes.Buffer, value any) {
	servers, ok := value.([]interface{})
	if !ok {
		return
	}
	
	for _, s := range servers {
		serverMap, ok := s.(map[string]any)
		if !ok {
			continue
		}
		
		id := serverMap["id"]
		host := serverMap["host"]
		peerPort := serverMap["peerPort"]
		electionPort := serverMap["electionPort"]
		
		buf.WriteString(fmt.Sprintf("server.%v=%v:%v:%v\n", id, host, peerPort, electionPort))
	}
}

// writeNested handles nested maps with dot notation
func writeNested(buf *bytes.Buffer, prefix string, nested map[string]any) {
	keys := make([]string, 0, len(nested))
	for k := range nested {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	for _, k := range keys {
		buf.WriteString(fmt.Sprintf("%s.%s=%v\n", prefix, k, nested[k]))
	}
}