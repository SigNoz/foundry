package ecsterraformcasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.MoldingEnricher = (*ecsMoldingEnricher)(nil)

const (
	telemetryStorePort        = 9000
	telemetryKeeperClientPort = 9181
	telemetryKeeperRaftPort   = 9234
	zookeeperClientPort       = 2181
	zookeeperRaftPort         = 2888
	metaStorePort             = 5432
	signozAPIServerPort       = 8080
	signozOpampPort           = 4320
	mcpHTTPPort               = 8000
)

const (
	// sdNamesPath selects every Cloud Map service name a module renders, in the
	// order the template emits them. The enricher reads names back rather than
	// recomputing them, so the template stays the single source of node names
	// and ordering (no drift).
	sdNamesPath   = "resource.aws_service_discovery_service.@values.#.name"
	namespacePath = "resource.aws_service_discovery_private_dns_namespace.main.name"
)

type ecsMoldingEnricher struct {
	materials []domain.StructuredMaterial
}

func newEcsMoldingEnricher(data templateData) (*ecsMoldingEnricher, error) {
	materials, err := getMaterials(data)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to render materials")
	}

	return &ecsMoldingEnricher{materials: materials}, nil
}

func (e *ecsMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error {
	ns, err := e.materials[0].GetBytes(namespacePath)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to read service discovery namespace")
	}

	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		names, err := e.materials[1].GetStringSlice(sdNamesPath)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to read telemetrystore service names")
		}

		addresses := make([]string, 0, len(names))
		for _, name := range names {
			host := name + "." + string(ns)
			addresses = append(addresses, domain.MustNewAddress("tcp", host, telemetryStorePort).String())
		}

		config.Spec.TelemetryStore.Status.Addresses.TCP = addresses

	case v1alpha1.MoldingKindTelemetryKeeper:
		names, err := e.materials[2].GetStringSlice(sdNamesPath)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to read telemetrykeeper service names")
		}

		clientPort, raftPort := telemetryKeeperClientPort, telemetryKeeperRaftPort
		if config.Spec.TelemetryKeeper.Kind == installation.TelemetryKeeperKindZookeeper {
			clientPort, raftPort = zookeeperClientPort, zookeeperRaftPort
		}

		clientAddresses := make([]string, 0, len(names))
		raftAddresses := make([]string, 0, len(names))
		for _, name := range names {
			host := name + "." + string(ns)
			clientAddresses = append(clientAddresses, domain.MustNewAddress("tcp", host, clientPort).String())
			raftAddresses = append(raftAddresses, domain.MustNewAddress("tcp", host, raftPort).String())
		}

		config.Spec.TelemetryKeeper.Status.Addresses.Client = clientAddresses
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = raftAddresses

	case v1alpha1.MoldingKindMetaStore:
		if config.Spec.MetaStore.Kind != installation.MetaStoreKindPostgres {
			return nil
		}

		names, err := e.materials[3].GetStringSlice(sdNamesPath)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to read metastore service names")
		}

		if len(names) > 0 {
			host := names[0] + "." + string(ns)
			config.Spec.MetaStore.Status.Addresses.DSN = []string{domain.MustNewAddress("tcp", host, metaStorePort).String()}
		}

	case v1alpha1.MoldingKindSignoz:
		names, err := e.materials[4].GetStringSlice(sdNamesPath)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to read signoz service names")
		}

		if len(names) > 0 {
			host := names[0] + "." + string(ns)
			config.Spec.Signoz.Status.Addresses.APIServer = []string{domain.MustNewAddress("tcp", host, signozAPIServerPort).String()}
			config.Spec.Signoz.Status.Addresses.Opamp = []string{domain.MustNewAddress("ws", host, signozOpampPort).String()}
		}

	case v1alpha1.MoldingKindMCP:
		if !config.Spec.MCP.Spec.IsEnabled() {
			return nil
		}

		names, err := e.materials[6].GetStringSlice(sdNamesPath)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to read mcp service names")
		}

		if len(names) > 0 {
			host := names[0] + "." + string(ns)
			config.Spec.MCP.Status.Addresses.HTTP = []string{domain.MustNewAddress("http", host, mcpHTTPPort).String()}
		}
	}

	return nil
}
