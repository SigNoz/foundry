package ecsterraformcasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

const (
	telemetryStorePort        = 9000
	telemetryKeeperClientPort = 9181
	telemetryKeeperRaftPort   = 9234
	metaStorePort             = 5432
	signozAPIPort             = 8080
	signozOpampPort           = 4320
)

var _ molding.MoldingEnricher = (*ecsMoldingEnricher)(nil)

type ecsMoldingEnricher struct {
	materials []types.Material
}

func newEcsMoldingEnricher(config *v1alpha1.Casting) (*ecsMoldingEnricher, error) {
	materials, err := getMaterials(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get materials: %w", err)
	}

	return &ecsMoldingEnricher{materials: materials}, nil
}

func (enricher *ecsMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	namespaceBytes, err := enricher.materials[0].GetBytes("resource.aws_service_discovery_private_dns_namespace.main.name")
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}
	namespace := string(namespaceBytes)

	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		sdName, err := enricher.materials[1].GetBytes("resource.aws_service_discovery_service.telemetrystore.name")
		if err != nil {
			return fmt.Errorf("failed to get telemetrystore service discovery name: %w", err)
		}
		fqdn := fmt.Sprintf("%s.%s", string(sdName), namespace)
		config.Spec.TelemetryStore.Status.Addresses.TCP = []string{types.FormatAddress("tcp", fqdn, telemetryStorePort)}

	case v1alpha1.MoldingKindTelemetryKeeper:
		sdName, err := enricher.materials[2].GetBytes("resource.aws_service_discovery_service.telemetrykeeper.name")
		if err != nil {
			return fmt.Errorf("failed to get telemetrykeeper service discovery name: %w", err)
		}
		fqdn := fmt.Sprintf("%s.%s", string(sdName), namespace)
		config.Spec.TelemetryKeeper.Status.Addresses.Client = []string{types.FormatAddress("tcp", fqdn, telemetryKeeperClientPort)}
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = []string{types.FormatAddress("tcp", fqdn, telemetryKeeperRaftPort)}

	case v1alpha1.MoldingKindMetaStore:
		sdName, err := enricher.materials[3].GetBytes("resource.aws_service_discovery_service.metastore.name")
		if err != nil {
			return fmt.Errorf("failed to get metastore service discovery name: %w", err)
		}
		fqdn := fmt.Sprintf("%s.%s", string(sdName), namespace)
		config.Spec.MetaStore.Status.Addresses.DSN = []string{types.FormatAddress("tcp", fqdn, metaStorePort)}

	case v1alpha1.MoldingKindSignoz:
		sdName, err := enricher.materials[4].GetBytes("resource.aws_service_discovery_service.signoz.name")
		if err != nil {
			return fmt.Errorf("failed to get signoz service discovery name: %w", err)
		}
		fqdn := fmt.Sprintf("%s.%s", string(sdName), namespace)
		config.Spec.Signoz.Status.Addresses.APIServer = []string{types.FormatAddress("tcp", fqdn, signozAPIPort)}
		config.Spec.Signoz.Status.Addresses.Opamp = []string{types.FormatAddress("ws", fqdn, signozOpampPort)}
	}

	return nil
}
