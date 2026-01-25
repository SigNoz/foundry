package telemetrykeepermolding

import (
	"embed"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	KeeperClickhousev2556YAML *types.Template = types.MustNewTemplateFromFS(templates, "templates/keeper.clickhouse.v2556.yaml.gotmpl", types.FormatYAML)
)

// Data is the template data for rendering ClickHouse Keeper configs.
type Data struct {
	RaftAddresses   []types.Address // Inter-keeper consensus addresses
	ClientAddresses []types.Address // Client-facing addresses
	ServerCount     int
	ServerID        int // Current server ID for per-node config generation
}

func newData(config *v1alpha1.Casting) (Data, error) {
	var data Data

	if config.Spec.TelemetryKeeper.Spec.Cluster.Replicas == nil {
		data.ServerCount = 1
	} else {
		data.ServerCount = max(*config.Spec.TelemetryKeeper.Spec.Cluster.Replicas, 1)
	}

	raftAddresses := config.Spec.TelemetryKeeper.Status.Addresses.Raft
	if len(raftAddresses) < data.ServerCount {
		return Data{}, fmt.Errorf("insufficient raft addresses: have %d, need %d servers", len(raftAddresses), data.ServerCount)
	}

	clientAddresses := config.Spec.TelemetryKeeper.Status.Addresses.Client
	if len(clientAddresses) < data.ServerCount {
		return Data{}, fmt.Errorf("insufficient client addresses: have %d, need %d servers", len(clientAddresses), data.ServerCount)
	}

	newRaftAddrs, err := types.NewAddresses(raftAddresses[:data.ServerCount])
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse raft addresses: %w", err)
	}
	data.RaftAddresses = newRaftAddrs

	newClientAddrs, err := types.NewAddresses(clientAddresses[:data.ServerCount])
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse client addresses: %w", err)
	}
	data.ClientAddresses = newClientAddrs

	return data, nil
}
