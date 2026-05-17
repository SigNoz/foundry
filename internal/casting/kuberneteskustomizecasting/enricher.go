package kuberneteskustomizecasting

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
	telemetryKeeperClientPort = 9181
	telemetryKeeperRaftPort   = 9234
	signozOpampPort           = 4320
)

var _ molding.MoldingEnricher = (*kustomizeMoldingEnricher)(nil)

type kustomizeMoldingEnricher struct {
	materials         []domain.StructuredMaterial
	overrideMaterials []domain.StructuredMaterial
}

func newKustomizeMoldingEnricher(config *installation.Casting) (*kustomizeMoldingEnricher, error) {
	materials, err := getServiceMaterials(config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to get service yaml material")
	}

	overrideMaterials, err := getOverrideMaterials(config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to get override materials")
	}

	return &kustomizeMoldingEnricher{
		materials:         materials,
		overrideMaterials: overrideMaterials,
	}, nil
}

func (e *kustomizeMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error {
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

func (e *kustomizeMoldingEnricher) enrichTelemetryStore(config *installation.Casting) error {
	name, err := e.materials[0].GetBytes("spec.templates.serviceTemplates.0.generateName")
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to get telemetrystore service names")
	}
	config.Spec.TelemetryStore.Status.Addresses.TCP = []string{domain.MustNewAddress("tcp", string(name), telemetryStorePort).String()}

	if config.Spec.TelemetryStore.Status.Extras == nil {
		config.Spec.TelemetryStore.Status.Extras = make(map[string]string)
	}
	config.Spec.TelemetryStore.Status.Extras["_overrides"] = string(e.overrideMaterials[0].FmtContents())

	return nil
}

func (e *kustomizeMoldingEnricher) enrichTelemetryKeeper(config *installation.Casting) error {
	spec := &config.Spec.TelemetryKeeper
	replicas := 1
	if spec.Spec.Cluster.Replicas != nil && *spec.Spec.Cluster.Replicas > 0 {
		replicas = *spec.Spec.Cluster.Replicas
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
	config.Spec.TelemetryKeeper.Status.Addresses.Client = client
	config.Spec.TelemetryKeeper.Status.Addresses.Raft = raft
	return nil
}

func (e *kustomizeMoldingEnricher) enrichMetaStore(config *installation.Casting) error {
	name, err := e.materials[1].GetBytes("metadata.name")
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to get metastore service names")
	}
	config.Spec.MetaStore.Status.Addresses.DSN = []string{
		fmt.Sprintf("postgres://%s:5432", name),
	}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichSignoz(config *installation.Casting) error {
	name := config.Metadata.Name + "-signoz"
	config.Spec.Signoz.Status.Addresses.Opamp = []string{domain.MustNewAddress("ws", name, signozOpampPort).String()}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichIngester(config *installation.Casting) error {
	// No-op: ingester molding only writes Status.Config.Data from other status.
	return nil
}
