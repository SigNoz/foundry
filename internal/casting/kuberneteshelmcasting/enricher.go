package kuberneteshelmcasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*helmMoldingEnricher)(nil)

type helmMoldingEnricher struct {
	materials []types.Material
}

func newHelmMoldingEnricher(_ *v1alpha1.Casting) *helmMoldingEnricher {
	return &helmMoldingEnricher{materials: []types.Material{}}
}

func (e *helmMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		return e.enrichTelemetryStore(config)
	case v1alpha1.MoldingKindTelemetryKeeper:
		return e.enrichTelemetryKeeper(config)
	case v1alpha1.MoldingKindMetaStore:
		return e.enrichMetaStore(config)
	case v1alpha1.MoldingKindSignoz:
		return e.enrichSignoz(config)
	case v1alpha1.MoldingKindIngester:
		return e.enrichIngester(config)
	}
	return nil
}

func (e *helmMoldingEnricher) enrichTelemetryStore(config *v1alpha1.Casting) error {
	name := fmt.Sprintf("%s-telemetrystore-%s", config.Metadata.Name, config.Spec.TelemetryStore.Kind)
	config.Spec.TelemetryStore.Status.Addresses.TCP = []string{types.FormatAddress("tcp", name, 9000)}
	return nil
}

func (e *helmMoldingEnricher) enrichTelemetryKeeper(config *v1alpha1.Casting) error {
	spec := &config.Spec.TelemetryKeeper
	replicas := 1
	if spec.Spec.Cluster.Replicas != nil && *spec.Spec.Cluster.Replicas > 0 {
		replicas = *spec.Spec.Cluster.Replicas
	}
	if replicas < 1 {
		replicas = 1
	}
	// Hardcoded to "zookeeper" because the chart deploys zookeeper, not clickhousekeeper.
	base := fmt.Sprintf("%s-telemetrykeeper-zookeeper", config.Metadata.Name)
	var client, raft []string
	for i := 0; i < replicas; i++ {
		client = append(client, types.FormatAddress("tcp", fmt.Sprintf("%s-%d", base, i), 9181))
		raft = append(raft, types.FormatAddress("tcp", fmt.Sprintf("%s-%d", base, i), 9234))
	}
	config.Spec.TelemetryKeeper.Status.Addresses.Client = client
	config.Spec.TelemetryKeeper.Status.Addresses.Raft = raft
	return nil
}

func (e *helmMoldingEnricher) enrichMetaStore(config *v1alpha1.Casting) error {
	name := fmt.Sprintf("%s-metastore-%s", config.Metadata.Name, config.Spec.MetaStore.Kind)
	config.Spec.MetaStore.Status.Addresses.DSN = []string{
		fmt.Sprintf("postgres://%s:5432", name),
	}
	return nil
}

func (e *helmMoldingEnricher) enrichSignoz(config *v1alpha1.Casting) error {
	// Chart uses signoz.fullname which resolves to fullnameOverride directly.
	name := config.Metadata.Name
	config.Spec.Signoz.Status.Addresses.Opamp = []string{types.FormatAddress("tcp", name, 4320)}
	return nil
}

func (e *helmMoldingEnricher) enrichIngester(config *v1alpha1.Casting) error {
	// No-op: ingester molding only writes Status.Config.Data from other status.
	return nil
}
