package dockerswarmcasting

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

var _ molding.MoldingEnricher = (*dockerSwarmMoldingEnricher)(nil)

type dockerSwarmMoldingEnricher struct {
	material domain.StructuredMaterial
}

func newDockerSwarmMoldingEnricher(config *installation.Casting) (*dockerSwarmMoldingEnricher, error) {
	material, err := getComposeMaterial(config, filepath.Join(rootcasting.DeploymentDir, "compose.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to get compose yaml material")
	}

	return &dockerSwarmMoldingEnricher{material: material}, nil
}

func (enricher *dockerSwarmMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to get telemetrystore service names")
		}

		var telemetrystoreAddresses []string
		for _, name := range containerNames {
			if strings.Contains(name, "telemetrystore-clickhouse") && !strings.Contains(name, "user-scripts") {
				telemetrystoreAddresses = append(telemetrystoreAddresses, domain.MustNewAddress("tcp", name, 9000).String())
			}
		}

		config.Spec.TelemetryStore.Status.Addresses.TCP = telemetrystoreAddresses

	case v1alpha1.MoldingKindSignoz:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to get signoz service names")
		}

		var apiServerAddr []string
		var opampAddr []string
		for _, name := range containerNames {
			if strings.Contains(name, "-signoz") {
				apiServerAddr = append(apiServerAddr, domain.MustNewAddress("tcp", name, 8080).String())
				opampAddr = append(opampAddr, domain.MustNewAddress("ws", name, 4320).String())
			}
		}
		config.Spec.Signoz.Status.Addresses.APIServer = apiServerAddr
		config.Spec.Signoz.Status.Addresses.Opamp = opampAddr

	case v1alpha1.MoldingKindTelemetryKeeper:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to get telemetrykeeper service names")
		}

		var clientAddresses []string
		for _, name := range containerNames {
			if strings.Contains(name, "telemetrykeeper") {
				clientAddresses = append(clientAddresses, domain.MustNewAddress("tcp", name, 9181).String())
			}
		}
		config.Spec.TelemetryKeeper.Status.Addresses.Client = clientAddresses

		var raftAddresses []string
		for _, name := range containerNames {
			if strings.Contains(name, "telemetrykeeper") {
				raftAddresses = append(raftAddresses, domain.MustNewAddress("tcp", name, 9234).String())
			}
		}
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = raftAddresses

	case v1alpha1.MoldingKindMetaStore:
		containerNames, err := enricher.material.GetStringSlice("services|@keys")
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to get metastore service names")
		}

		var metastoreAddresses []string
		for _, name := range containerNames {
			if strings.Contains(name, "metastore") {
				metastoreAddresses = append(metastoreAddresses, domain.MustNewAddress("tcp", name, 5432).String())
			}
		}
		config.Spec.MetaStore.Status.Addresses.DSN = metastoreAddresses

	case v1alpha1.MoldingKindIngester:
		// The ingester is scaled via `deploy.replicas` and reached through the
		// `<metadata.name>-ingester` network alias, which Swarm's routing mesh
		// load-balances across all replicas.
		config.Spec.Ingester.Status.Addresses.OTLP = []string{
			domain.MustNewAddress("tcp", config.Metadata.Name+"-ingester", 4318).String(),
			domain.MustNewAddress("tcp", config.Metadata.Name+"-ingester", 4317).String(),
		}
	}

	return nil
}
