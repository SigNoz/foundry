package dockercomposecasting

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.MoldingEnricher = (*dockerComposeMoldingEnricher)(nil)

type dockerComposeMoldingEnricher struct {
	material domain.StructuredMaterial
}

func newDockerComposeMoldingEnricher(config *installation.Casting) (*dockerComposeMoldingEnricher, error) {
	material, err := getComposeMaterial(config, filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to get compose yaml material")
	}

	return &dockerComposeMoldingEnricher{material: material}, nil
}

func (enricher *dockerComposeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		// Get telemetrystore container names
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to get telemetrystore container names")
		}

		var telemetrystoreContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "telemetrystore-clickhouse") && !strings.Contains(containerName, "user-scripts") {
				telemetrystoreContainerNames = append(telemetrystoreContainerNames, domain.MustNewAddress("tcp", containerName, 9000).String())
			}
		}

		config.Spec.TelemetryStore.Status.Addresses.TCP = telemetrystoreContainerNames

	case v1alpha1.MoldingKindSignoz:
		// Get signoz container names
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to get signoz container names")
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
		// Get telemetrykeeper container names (using service keys since they match container_name)
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to get telemetrykeeper container names")
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
		// Skip molding enrichment if sqlite
		if config.Spec.MetaStore.Kind == installation.MetaStoreKindSQLite {
			return nil
		}
		// Get metastore container names
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to get metastore container names")
		}

		var metastoreContainerNames []string
		for _, containerName := range containerNames {
			if strings.Contains(containerName, "metastore") {
				metastoreContainerNames = append(metastoreContainerNames, domain.MustNewAddress("tcp", containerName, 5432).String())
			}
		}

		config.Spec.MetaStore.Status.Addresses.DSN = metastoreContainerNames

	case v1alpha1.MoldingKindIngester:
		// The ingester is scaled via `deploy.replicas` and reached through the
		// `<metadata.name>-ingester` network alias, which compose load-balances
		// across all replicas.
		config.Spec.Ingester.Status.Addresses.OTLP = []string{
			domain.MustNewAddress("tcp", config.Metadata.Name+"-ingester", 4318).String(),
			domain.MustNewAddress("tcp", config.Metadata.Name+"-ingester", 4317).String(),
		}
	}

	return nil
}
