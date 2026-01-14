package dockercomposecasting

import (
	"context"
	"fmt"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*dockerComposeMoldingEnricher)(nil)

type dockerComposeMoldingEnricher struct {
	material types.Material
}

func newDockerComposeMoldingEnricher(config *v1alpha1.Casting) (*dockerComposeMoldingEnricher, error) {
	material, err := getComposeMaterial(config, "compose.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to get compose yaml material: %w", err)
	}

	return &dockerComposeMoldingEnricher{material: material}, nil
}

func (enricher *dockerComposeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		// Get telemetrystore container names
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get telemetrystore container names: %w", err)
		}

		var telemetrystoreContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrystore") && !strings.Contains(containerName, "user-scripts") {
				telemetrystoreContainerNames = append(telemetrystoreContainerNames, types.FormatAddress("tcp", containerName, 9000))
			}
		}

		if config.Spec.TelemetryStore.Status.Addresses == nil {
			config.Spec.TelemetryStore.Status.Addresses = make(map[string][]string)
		}
		config.Spec.TelemetryStore.Status.Addresses[v1alpha1.TelemetryStoreClusterAddresses] = telemetrystoreContainerNames

	case v1alpha1.MoldingKindSignoz:
		// Get signoz container names
		containerNames, err := enricher.material.GetStringSlice("services.*.container_name")
		if err != nil {
			return fmt.Errorf("failed to get signoz container names: %w", err)
		}

		var signozContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "signoz") {
				signozContainerNames = append(signozContainerNames, types.FormatAddress("tcp", containerName, 9000))
			}
		}

		if config.Spec.Signoz.Status.Addresses == nil {
			config.Spec.Signoz.Status.Addresses = make(map[string][]string)
		}
		config.Spec.Signoz.Status.Addresses[v1alpha1.SignozAPIAddresses] = signozContainerNames

	case v1alpha1.MoldingKindTelemetryKeeper:
		// Get telemetrykeeper container names
		containerNames, err := enricher.material.GetStringSlice("services.*.container_name")
		if err != nil {
			return fmt.Errorf("failed to get telemetrykeeper container names: %w", err)
		}

		var telemetrykeeperContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrykeeper") {
				telemetrykeeperContainerNames = append(telemetrykeeperContainerNames, types.FormatAddress("tcp", containerName, 9181))
			}
		}

		if config.Spec.TelemetryKeeper.Status.Addresses == nil {
			config.Spec.TelemetryKeeper.Status.Addresses = make(map[string][]string)
		}

		config.Spec.TelemetryKeeper.Status.Addresses[v1alpha1.TelemetryKeeperClientAddresses] = telemetrykeeperContainerNames

		var telemetryRaftaddress []string
		for _, containerName := range containerNames {
			telemetryRaftaddress = append(telemetryRaftaddress, types.FormatAddress("tcp", containerName, 9234))
		}
		config.Spec.TelemetryKeeper.Status.Addresses[v1alpha1.TelemetryKeeperRaftAddresses] = telemetryRaftaddress

	case v1alpha1.MoldingKindMetaStore:
		// Get metastore container names
		containerNames, err := enricher.material.GetStringSlice("services.*.container_name")
		if err != nil {
			return fmt.Errorf("failed to get metastore container names: %w", err)
		}

		var metastoreContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "metastore") {
				metastoreContainerNames = append(metastoreContainerNames, types.FormatAddress("tcp", containerName, 9000))
			}
		}

		if config.Spec.MetaStore.Status.Addresses == nil {
			config.Spec.MetaStore.Status.Addresses = make(map[string][]string)
		}
		config.Spec.MetaStore.Status.Addresses[v1alpha1.MetaStoreDSNAddresses] = metastoreContainerNames

	case v1alpha1.MoldingKindIngester:
		// Get ingester container names
		containerNames, err := enricher.material.GetStringSlice("services.*.container_name")
		if err != nil {
			return fmt.Errorf("failed to get ingester container names: %w", err)
		}

		var ingesterContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "ingester") {
				ingesterContainerNames = append(ingesterContainerNames, types.FormatAddress("tcp", containerName, 9000))
			}
		}

		if config.Spec.Ingester.Status.Addresses == nil {
			config.Spec.Ingester.Status.Addresses = make(map[string][]string)
		}
		config.Spec.Ingester.Status.Addresses[v1alpha1.IngesterReceiverAddresses] = ingesterContainerNames
	}

	return nil
}
