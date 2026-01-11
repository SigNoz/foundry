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
				telemetrystoreContainerNames = append(telemetrystoreContainerNames, types.NewAddress("tcp", containerName, 9000))
			}
		}

		config.Spec.TelemetryStore.Status.Addresses = telemetrystoreContainerNames

	case v1alpha1.MoldingKindSignoz:
		// Get signoz container names
		containerNames, err := enricher.material.GetStringSlice("services.*.container_name")
		if err != nil {
			return fmt.Errorf("failed to get signoz container names: %w", err)
		}

		var signozContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "signoz") {
				signozContainerNames = append(signozContainerNames, types.NewAddress("tcp", containerName, 9000))
			}
		}

		config.Spec.Signoz.Status.Addresses = signozContainerNames

	case v1alpha1.MoldingKindTelemetryKeeper:
		// Get telemetrykeeper container names
		containerNames, err := enricher.material.GetStringSlice("services.*.container_name")
		if err != nil {
			return fmt.Errorf("failed to get telemetrykeeper container names: %w", err)
		}

		var telemetrykeeperContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrykeeper") {
				telemetrykeeperContainerNames = append(telemetrykeeperContainerNames, types.NewAddress("tcp", containerName, 9000))
			}
		}

		config.Spec.TelemetryKeeper.Status.Addresses = telemetrykeeperContainerNames

	case v1alpha1.MoldingKindMetaStore:
		// Get metastore container names
		containerNames, err := enricher.material.GetStringSlice("services.*.container_name")
		if err != nil {
			return fmt.Errorf("failed to get metastore container names: %w", err)
		}

		var metastoreContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "metastore") {
				metastoreContainerNames = append(metastoreContainerNames, types.NewAddress("tcp", containerName, 9000))
			}
		}

		config.Spec.MetaStore.Status.Addresses = metastoreContainerNames

	case v1alpha1.MoldingKindIngester:
		// Get ingester container names
		containerNames, err := enricher.material.GetStringSlice("services.*.container_name")
		if err != nil {
			return fmt.Errorf("failed to get ingester container names: %w", err)
		}

		var ingesterContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "ingester") {
				ingesterContainerNames = append(ingesterContainerNames, types.NewAddress("tcp", containerName, 9000))
			}
		}

		config.Spec.Ingester.Status.Addresses = ingesterContainerNames
	}

	return nil
}
