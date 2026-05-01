package coolifycasting

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.MoldingEnricher = (*coolifyMoldingEnricher)(nil)

type coolifyMoldingEnricher struct {
	material domain.StructuredMaterial
}

func newCoolifyMoldingEnricher(config *v1alpha1.Casting) (*coolifyMoldingEnricher, error) {
	material, err := getCoolifyMaterial(config, filepath.Join(rootcasting.DeploymentDir, "coolify.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to get coolify yaml material: %w", err)
	}
	return &coolifyMoldingEnricher{material: material}, nil
}

func (enricher *coolifyMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get telemetrystore container names: %w", err)
		}

		var telemetrystoreContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrystore-clickhouse") && !strings.Contains(containerName, "user-scripts") {
				telemetrystoreContainerNames = append(telemetrystoreContainerNames, domain.MustNewAddress("tcp", containerName, 9000).String())
			}
		}
		config.Spec.TelemetryStore.Status.Addresses.TCP = telemetrystoreContainerNames

	case v1alpha1.MoldingKindSignoz:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get signoz container names: %w", err)
		}

		var apiServerAddr []string
		var opampAddr []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "-signoz") {
				apiServerAddr = append(apiServerAddr, domain.MustNewAddress("tcp", containerName, 8080).String())
				opampAddr = append(opampAddr, domain.MustNewAddress("ws", containerName, 4320).String())
			}
		}
		config.Spec.Signoz.Status.Addresses.APIServer = apiServerAddr
		config.Spec.Signoz.Status.Addresses.Opamp = opampAddr

	case v1alpha1.MoldingKindTelemetryKeeper:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get telemetrykeeper container names: %w", err)
		}

		var telemetrykeeperContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrykeeper") {
				telemetrykeeperContainerNames = append(telemetrykeeperContainerNames, domain.MustNewAddress("tcp", containerName, 9181).String())
			}
		}
		config.Spec.TelemetryKeeper.Status.Addresses.Client = telemetrykeeperContainerNames

		var telemetryRaftaddress []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrykeeper") {
				telemetryRaftaddress = append(telemetryRaftaddress, domain.MustNewAddress("tcp", containerName, 9234).String())
			}
		}
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = telemetryRaftaddress

	case v1alpha1.MoldingKindMetaStore:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get metastore container names: %w", err)
		}

		var metastoreContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "metastore") {
				metastoreContainerNames = append(metastoreContainerNames, domain.MustNewAddress("tcp", containerName, 5432).String())
			}
		}
		config.Spec.MetaStore.Status.Addresses.DSN = metastoreContainerNames

	case v1alpha1.MoldingKindIngester:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return fmt.Errorf("failed to get ingester container names: %w", err)
		}

		var ingesterContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "ingester") {
				ingesterContainerNames = append(ingesterContainerNames, domain.MustNewAddress("tcp", containerName, 4318).String())
			}
		}
		config.Spec.Ingester.Status.Addresses.OTLP = ingesterContainerNames
	}

	return nil
}
