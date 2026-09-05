package kuberneteshelmcasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

const (
	telemetryStorePort        = 9000
	telemetryKeeperClientPort = 2181
	telemetryKeeperRaftPort   = 2888
	metaStorePort             = 5432
	signozAPIServerPort       = 8080
	signozOpampPort           = 4320
	ingesterOTLPGRPCPort      = 4317
	ingesterOTLPHTTPPort      = 4318
)

var _ molding.MoldingEnricher = (*helmMoldingEnricher)(nil)

type helmMoldingEnricher struct{}

func newHelmMoldingEnricher(_ *installation.Casting) *helmMoldingEnricher {
	return &helmMoldingEnricher{}
}

// EnrichStatus refuses what the chart has no values for.
func (e *helmMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error {
	deployment := config.Spec.Deployment

	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		if !config.Spec.TelemetryStore.Spec.IsEnabled() {
			return errors.Newf(errors.TypeUnsupported, "deployment '%s/%s' does not support telemetrystore.spec.enabled: false, the chart then requires externalClickhouse which foundry does not translate, raise an issue at https://github.com/signoz/foundry/issues", deployment.Mode, deployment.Flavor)
		}

		name := fmt.Sprintf("%s-telemetrystore-%s", config.Metadata.Name, config.Spec.TelemetryStore.Kind)
		config.Spec.TelemetryStore.Status.Addresses.TCP = []string{domain.MustNewAddress("tcp", name, telemetryStorePort).String()}

	case v1alpha1.MoldingKindTelemetryKeeper:
		spec := &config.Spec.TelemetryKeeper
		if spec.Kind != installation.TelemetryKeeperKindZookeeper {
			return errors.Newf(errors.TypeUnsupported, "deployment '%s/%s' does not support telemetrykeeper.kind: %s, the chart ships zookeeper only, set telemetrykeeper.kind: zookeeper or raise an issue at https://github.com/signoz/foundry/issues", deployment.Mode, deployment.Flavor, spec.Kind)
		}

		replicas := 1
		if spec.Spec.Cluster.Replicas != nil && *spec.Spec.Cluster.Replicas > 1 {
			replicas = *spec.Spec.Cluster.Replicas
		}

		base := fmt.Sprintf("%s-telemetrykeeper-zookeeper", config.Metadata.Name)

		var client, raft []string
		for i := 0; i < replicas; i++ {
			client = append(client, domain.MustNewAddress("tcp", fmt.Sprintf("%s-%d", base, i), telemetryKeeperClientPort).String())
			raft = append(raft, domain.MustNewAddress("tcp", fmt.Sprintf("%s-%d", base, i), telemetryKeeperRaftPort).String())
		}

		spec.Status.Addresses.Client = client
		spec.Status.Addresses.Raft = raft

	case v1alpha1.MoldingKindMetaStore:
		if !config.Spec.MetaStore.Spec.IsEnabled() || config.Spec.MetaStore.Kind != installation.MetaStoreKindPostgres {
			return nil
		}

		name := fmt.Sprintf("%s-metastore-%s", config.Metadata.Name, config.Spec.MetaStore.Kind)
		config.Spec.MetaStore.Status.Addresses.DSN = []string{fmt.Sprintf("postgres://%s:%d", name, metaStorePort)}

	case v1alpha1.MoldingKindSignoz:
		// signoz.fullname is the fullnameOverride verbatim: no component suffix.
		name := config.Metadata.Name
		config.Spec.Signoz.Status.Addresses.APIServer = []string{domain.MustNewAddress("tcp", name, signozAPIServerPort).String()}
		config.Spec.Signoz.Status.Addresses.Opamp = []string{domain.MustNewAddress("ws", name, signozOpampPort).String()}

	case v1alpha1.MoldingKindIngester:
		if !config.Spec.Ingester.Spec.IsEnabled() {
			return errors.Newf(errors.TypeUnsupported, "deployment '%s/%s' does not support ingester.spec.enabled: false, the chart renders the collector unconditionally, raise an issue at https://github.com/signoz/foundry/issues", deployment.Mode, deployment.Flavor)
		}

		name := config.Metadata.Name + "-ingester"
		config.Spec.Ingester.Status.Addresses.OTLP = []string{
			domain.MustNewAddress("tcp", name, ingesterOTLPHTTPPort).String(),
			domain.MustNewAddress("tcp", name, ingesterOTLPGRPCPort).String(),
		}

	case v1alpha1.MoldingKindMCP:
		if config.Spec.MCP.Spec.IsEnabled() {
			return errors.Newf(errors.TypeUnsupported, "deployment '%s/%s' does not support mcp yet, the chart has no mcp component, raise an issue at https://github.com/signoz/foundry/issues", deployment.Mode, deployment.Flavor)
		}
	}

	return nil
}
