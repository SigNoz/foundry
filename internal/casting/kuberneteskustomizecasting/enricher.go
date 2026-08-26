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
	signozAPIServerPort       = 8080
	mcpHTTPPort               = 8000
	ingesterOTLPGRPCPort      = 4317
	ingesterOTLPHTTPPort      = 4318
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
	case v1alpha1.MoldingKindMCP:
		return e.enrichMCP(config)
	}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichTelemetryStore(config *installation.Casting) error {
	name, err := e.materials[0].GetBytes("spec.templates.serviceTemplates.0.generateName")
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to get telemetrystore service names")
	}
	cluster := config.Spec.TelemetryStore.Spec.Cluster
	shards, replicas := 1, 1
	if cluster.Shards != nil && *cluster.Shards > 0 {
		shards = *cluster.Shards
	}
	if cluster.Replicas != nil {
		replicas = *cluster.Replicas + 1
	}

	addresses := []string{domain.MustNewAddress("tcp", string(name), telemetryStorePort).String()}
	for s := 0; s < shards; s++ {
		for r := 0; r < replicas; r++ {
			if s == 0 && r == 0 {
				continue
			}
			host := fmt.Sprintf("chi-%s-cluster-%d-%d", name, s, r)
			addresses = append(addresses, domain.MustNewAddress("tcp", host, telemetryStorePort).String())
		}
	}
	config.Spec.TelemetryStore.Status.Addresses.TCP = addresses

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
	if config.Spec.MetaStore.Kind == installation.MetaStoreKindSQLite {
		return nil
	}

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
	config.Spec.Signoz.Status.Addresses.APIServer = []string{domain.MustNewAddress("tcp", name, signozAPIServerPort).String()}
	config.Spec.Signoz.Status.Addresses.Opamp = []string{domain.MustNewAddress("ws", name, signozOpampPort).String()}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichMCP(config *installation.Casting) error {
	if !config.Spec.MCP.Spec.IsEnabled() {
		return nil
	}

	host := config.Metadata.Name + "-mcp." + config.Metadata.Name
	config.Spec.MCP.Status.Addresses.HTTP = []string{domain.MustNewAddress("http", host, mcpHTTPPort).String()}
	return nil
}

func (e *kustomizeMoldingEnricher) enrichIngester(config *installation.Casting) error {
	name := config.Metadata.Name + "-ingester"
	config.Spec.Ingester.Status.Addresses.OTLP = []string{
		domain.MustNewAddress("tcp", name, ingesterOTLPHTTPPort).String(),
		domain.MustNewAddress("tcp", name, ingesterOTLPGRPCPort).String(),
	}
	return nil
}
