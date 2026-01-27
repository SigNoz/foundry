package dockercomposecasting

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*dockerComposeMoldingEnricher)(nil)

type dockerComposeMoldingEnricher struct {
	material types.Material
}

func newDockerComposeMoldingEnricher(config *v1alpha1.Casting) (*dockerComposeMoldingEnricher, error) {
	material, err := getComposeMaterial(config, filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
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

		config.Spec.TelemetryStore.Status.Addresses.TCP = telemetrystoreContainerNames

	case v1alpha1.MoldingKindSignoz:
		// Get signoz container names
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get signoz container names: %w", err)
		}

		var signozContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "signoz") {
				signozContainerNames = append(signozContainerNames, types.FormatAddress("tcp", containerName, 9000))
			}
		}

		config.Spec.Signoz.Status.Addresses.APIServer = signozContainerNames
		config.Spec.Signoz.Status.Addresses.Opamp = signozContainerNames

	case v1alpha1.MoldingKindTelemetryKeeper:
		// Get telemetrykeeper container names (using service keys since they match container_name)
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get telemetrykeeper container names: %w", err)
		}

		var telemetrykeeperContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrykeeper") {
				telemetrykeeperContainerNames = append(telemetrykeeperContainerNames, types.FormatAddress("tcp", containerName, 9181))
			}
		}

		config.Spec.TelemetryKeeper.Status.Addresses.Client = telemetrykeeperContainerNames

		var telemetryRaftaddress []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrykeeper") {
				telemetryRaftaddress = append(telemetryRaftaddress, types.FormatAddress("tcp", containerName, 9234))
			}
		}

		config.Spec.TelemetryKeeper.Status.Addresses.Raft = telemetryRaftaddress

	case v1alpha1.MoldingKindMetaStore:
		// Get metastore container names
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get metastore container names: %w", err)
		}

		var metastoreContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "metastore") {
				metastoreContainerNames = append(metastoreContainerNames, types.FormatAddress("tcp", containerName, 9000))
			}
		}

		config.Spec.MetaStore.Status.Addresses.DSN = metastoreContainerNames

	case v1alpha1.MoldingKindIngester:
		// Get ingester container names
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get ingester container names: %w", err)
		}

		var ingesterContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "ingester") {
				ingesterContainerNames = append(ingesterContainerNames, types.FormatAddress("tcp", containerName, 9000))
			}
		}

		config.Spec.Ingester.Status.Addresses.OTLP = ingesterContainerNames
	}

	return nil
}
