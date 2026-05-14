package telemetrykeepermolding

import (
	"embed"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	KeeperClickhousev2556YAML *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/keeper.clickhouse.v2556.yaml.gotmpl", domain.FormatYAML)
)

// Data is the template data for rendering ClickHouse Keeper configs.
type Data struct {
	RaftAddresses   []domain.Address // Inter-keeper consensus addresses
	ClientAddresses []domain.Address // Client-facing addresses
	ServerCount     int
	ServerID        int // Current server ID for per-node config generation
}

func newData(config *installation.Casting) (Data, error) {
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

	newRaftAddrs, err := domain.ParseAddresses(raftAddresses[:data.ServerCount])
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse raft addresses: %w", err)
	}
	data.RaftAddresses = newRaftAddrs

	newClientAddrs, err := domain.ParseAddresses(clientAddresses[:data.ServerCount])
	if err != nil {
		return Data{}, fmt.Errorf("failed to parse client addresses: %w", err)
	}
	data.ClientAddresses = newClientAddrs

	return data, nil
}
