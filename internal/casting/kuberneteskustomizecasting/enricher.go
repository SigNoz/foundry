package kuberneteskustomizecasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
)

const (
	telemetryStorePort        = 9000
	telemetryKeeperClientPort = 9181
	telemetryKeeperRaftPort   = 9234
	signozOpampPort           = 4320
)

var _ molding.MoldingEnricher = (*kustomizeMoldingEnricher)(nil)

type kustomizeMoldingEnricher struct {
	materials         []domain.StructuredMaterial
	overrideMaterials []domain.StructuredMaterial
}

func newKustomizeMoldingEnricher(config *v1alpha1.Casting) (*kustomizeMoldingEnricher, error) {
	materials, err := getServiceMaterials(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get service yaml material: %w", err)
	}

	overrideMaterials, err := getOverrideMaterials(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get override materials: %w", err)
	}

	return &kustomizeMoldingEnricher{
		materials:         materials,
		overrideMaterials: overrideMaterials,
	}, nil
}

func (e *kustomizeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
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

func (e *kustomizeMoldingEnricher) enrichTelemetryStore(config *v1alpha1.Casting) error {
	spec := config.SigNozSpec()

	name, err := e.materials[0].GetBytes("spec.templates.serviceTemplates.0.generateName")
	if err != nil {
		return fmt.Errorf("failed to get telemetrystore service names: %w", err)
	}
	spec.TelemetryStore.Status.Addresses.TCP = []string{domain.MustNewAddress("tcp", string(name), telemetryStorePort).String()}

	if spec.TelemetryStore.Status.Extras == nil {
		spec.TelemetryStore.Status.Extras = make(map[string]string)
	}
	spec.TelemetryStore.Status.Extras["_overrides"] = string(e.overrideMaterials[0].FmtContents())

	return nil
}

func (e *kustomizeMoldingEnricher) enrichTelemetryKeeper(config *v1alpha1.Casting) error {
	spec := config.SigNozSpec()
	tk := &spec.TelemetryKeeper
	replicas := 1
	if tk.Spec.Cluster.Replicas != nil && *tk.Spec.Cluster.Replicas > 0 {
		replicas = *tk.Spec.Cluster.Replicas
	}
	if replicas < 1 {
		replicas = 1
	}
	// Dummy Variables, To pass validation in molding
	// TODO: Take the logic out of molding as operator handles it already
	base := config.Metadata.Name + "-clickhouse-keeper"
	var client, raft []string
	for i := 0; i < replicas; i++ {
		client = append(client, domain.MustNewAddress("tcp", fmt.Sprintf("%s-%d", base, i), telemetryKeeperClientPort).String())
		raft = append(raft, domain.MustNewAddress("tcp", fmt.Sprintf("%s-%d", base, i), telemetryKeeperRaftPort).String())
	}
	tk.Status.Addresses.Client = client
	tk.Status.Addresses.Raft = raft
	return nil
}

func (e *kustomizeMoldingEnricher) enrichMetaStore(config *v1alpha1.Casting) error {
	spec := config.SigNozSpec()

	name, err := e.materials[1].GetBytes("metadata.name")
	if err != nil {
		return fmt.Errorf("failed to get metastore service names: %w", err)
	}
	spec.MetaStore.Status.Addresses.DSN = []string{
		fmt.Sprintf("postgres://%s:5432", name),
	}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichSignoz(config *v1alpha1.Casting) error {
	spec := config.SigNozSpec()
	name := config.Metadata.Name + "-signoz"
	spec.Signoz.Status.Addresses.Opamp = []string{domain.MustNewAddress("ws", name, signozOpampPort).String()}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichIngester(config *v1alpha1.Casting) error {
	// No-op: ingester molding only writes Status.Config.Data from other status.
	return nil
}
