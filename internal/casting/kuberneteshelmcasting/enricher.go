package kuberneteshelmcasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.MoldingEnricher = (*helmMoldingEnricher)(nil)

type helmMoldingEnricher struct {
	materials []domain.Material
}

func newHelmMoldingEnricher(_ *v1alpha1.Casting) *helmMoldingEnricher {
	return &helmMoldingEnricher{materials: []domain.Material{}}
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
	spec := config.SigNozSpec()
	name := fmt.Sprintf("%s-telemetrystore-%s", config.Metadata.Name, spec.TelemetryStore.Kind)
	spec.TelemetryStore.Status.Addresses.TCP = []string{domain.MustNewAddress("tcp", name, 9000).String()}
	return nil
}

func (e *helmMoldingEnricher) enrichTelemetryKeeper(config *v1alpha1.Casting) error {
	spec := config.SigNozSpec()
	tk := &spec.TelemetryKeeper
	replicas := 1
	if tk.Spec.Cluster.Replicas != nil && *tk.Spec.Cluster.Replicas > 0 {
		replicas = *tk.Spec.Cluster.Replicas
	}
	if replicas < 1 {
		replicas = 1
	}
	// Hardcoded to "zookeeper" because the chart deploys zookeeper, not clickhousekeeper.
	base := fmt.Sprintf("%s-telemetrykeeper-zookeeper", config.Metadata.Name)
	var client, raft []string
	for i := 0; i < replicas; i++ {
		client = append(client, domain.MustNewAddress("tcp", fmt.Sprintf("%s-%d", base, i), 9181).String())
		raft = append(raft, domain.MustNewAddress("tcp", fmt.Sprintf("%s-%d", base, i), 9234).String())
	}
	tk.Status.Addresses.Client = client
	tk.Status.Addresses.Raft = raft
	return nil
}

func (e *helmMoldingEnricher) enrichMetaStore(config *v1alpha1.Casting) error {
	spec := config.SigNozSpec()
	name := fmt.Sprintf("%s-metastore-%s", config.Metadata.Name, spec.MetaStore.Kind)
	spec.MetaStore.Status.Addresses.DSN = []string{
		fmt.Sprintf("postgres://%s:5432", name),
	}
	return nil
}

func (e *helmMoldingEnricher) enrichSignoz(config *v1alpha1.Casting) error {
	spec := config.SigNozSpec()
	// Chart uses signoz.fullname which resolves to fullnameOverride directly.
	name := config.Metadata.Name
	spec.Signoz.Status.Addresses.Opamp = []string{domain.MustNewAddress("tcp", name, 4320).String()}
	return nil
}

func (e *helmMoldingEnricher) enrichIngester(config *v1alpha1.Casting) error {
	// No-op: ingester molding only writes Status.Config.Data from other status.
	return nil
}
