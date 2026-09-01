package systemdcasting

import (
	"context"
	"path"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.MoldingEnricher = (*systemdMoldingEnricher)(nil)

const (
	baseTelemetryKeeperClientPort          = 9181
	baseTelemetryKeeperRaftPort            = 9234
	baseTelemetryKeeperZookeeperClientPort = 2181
	baseTelemetryKeeperZookeeperRaftPort   = 2888
	baseTelemetryStoreClusterPort          = 9000
	baseMetaStorePostgresPort              = 5432
)

type systemdMoldingEnricher struct {
	materials []domain.Material
}

func newSystemdMoldingEnricher(config *installation.Casting) *systemdMoldingEnricher {
	// Record each annotation's resolved value so the lock captures the full
	// resolved config: user-set values win, absent ones fall back to the default.
	// The catalog holds every mode's; the rest would land in the lock as empty
	// keys that mean nothing here.
	if config.Metadata.Annotations == nil {
		config.Metadata.Annotations = map[string]string{}
	}
	for _, a := range installation.Annotations() {
		if a.Mode != config.Spec.Deployment.Mode {
			continue
		}

		config.Metadata.Annotations[a.Key] = a.Resolve(config.Metadata.Annotations)
	}

	return &systemdMoldingEnricher{materials: []domain.Material{}}
}

func (e *systemdMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryKeeper:
		return e.enrichTelemetryKeeper(config)
	case v1alpha1.MoldingKindTelemetryStore:
		return e.enrichTelemetryStore(config)
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

func (e *systemdMoldingEnricher) enrichTelemetryKeeper(config *installation.Casting) error {
	spec := &config.Spec.TelemetryKeeper
	cluster := spec.Spec.Cluster

	replicas := 1
	if cluster.Replicas != nil {
		replicas = max(*cluster.Replicas, 1)
	}

	if replicas > 1 {
		return errors.Newf(errors.TypeUnsupported, "deployment mode '%s' does not support Distributed Clickhouse Setup, raise an issue at https://github.com/signoz/foundry/issues", config.Spec.Deployment.Mode)
	}

	clientPort, raftPort := baseTelemetryKeeperClientPort, baseTelemetryKeeperRaftPort
	if spec.Kind == installation.TelemetryKeeperKindZookeeper {
		clientPort, raftPort = baseTelemetryKeeperZookeeperClientPort, baseTelemetryKeeperZookeeperRaftPort
	}

	var clientAddresses, raftAddresses []string
	for r := 0; r < replicas; r++ {
		clientAddresses = append(clientAddresses, domain.MustNewAddress("tcp", "localhost", clientPort+r).String())
		raftAddresses = append(raftAddresses, domain.MustNewAddress("tcp", "localhost", raftPort+r).String())
	}

	config.Spec.TelemetryKeeper.Status.Addresses.Client = clientAddresses
	config.Spec.TelemetryKeeper.Status.Addresses.Raft = raftAddresses
	return nil
}

func (e *systemdMoldingEnricher) enrichTelemetryStore(config *installation.Casting) error {
	spec := &config.Spec.TelemetryStore
	cluster := spec.Spec.Cluster

	replicas := 1
	shards := 1
	if cluster.Replicas != nil {
		replicas = max(*cluster.Replicas+1, 1)
	}
	if cluster.Shards != nil {
		shards = max(*cluster.Shards, 1)
	}

	if replicas > 1 || shards > 1 {
		return errors.Newf(errors.TypeUnsupported, "deployment mode '%s' does not support Distributed Clickhouse Setup, raise an issue at https://github.com/signoz/foundry/issues", config.Spec.Deployment.Mode)
	}

	// Generate addresses for each shard/replica
	var addresses []string
	for shard := 0; shard < shards; shard++ {
		for replica := 0; replica < replicas; replica++ {
			port := baseTelemetryStoreClusterPort + (shard * replicas) + replica
			addresses = append(addresses, domain.MustNewAddress("tcp", "localhost", port).String())
		}
	}

	config.Spec.TelemetryStore.Status.Addresses.TCP = addresses
	return nil
}

func (e *systemdMoldingEnricher) enrichMetaStore(config *installation.Casting) error {
	switch config.Spec.MetaStore.Kind {
	case installation.MetaStoreKindSQLite:
		// SQLite — no addresses or binaries to enrich.
	case installation.MetaStoreKindPostgres:
		dsn := domain.MustNewAddress("postgres", "localhost", baseMetaStorePostgresPort).String()
		config.Spec.MetaStore.Status.Addresses.DSN = []string{dsn}
	}
	return nil
}

func (e *systemdMoldingEnricher) enrichSignoz(config *installation.Casting) error {
	config.Spec.Signoz.Status.Addresses.Opamp = []string{
		domain.MustNewAddress("ws", "localhost", 4320).String(),
	}
	config.Spec.Signoz.Status.Addresses.APIServer = []string{
		domain.MustNewAddress("tcp", "localhost", 8080).String(),
	}

	// Resolve the signoz binary path from the catalog; its parent tree holds the
	// web and template assets the binary needs.
	signozBin := installation.SignozBinaryPath.Resolve(config.Metadata.Annotations)

	// The binary defaults these to its in-container paths, so point them at the
	// extracted tarball tree (binary lives at <root>/bin/signoz).
	root := path.Dir(path.Dir(signozBin))
	if config.Spec.Signoz.Status.Env == nil {
		config.Spec.Signoz.Status.Env = make(map[string]string)
	}
	env := config.Spec.Signoz.Status.Env
	env["SIGNOZ_WEB_DIRECTORY"] = path.Join(root, "web")
	env["SIGNOZ_EMAILING_TEMPLATES_DIRECTORY"] = path.Join(root, "templates", "email")
	env["SIGNOZ_ALERTMANAGER_SIGNOZ_TEMPLATES"] = path.Join(root, "templates", "alertmanager", "*.gotmpl")

	return nil
}

func (e *systemdMoldingEnricher) enrichIngester(config *installation.Casting) error {
	config.Spec.Ingester.Status.Addresses.OTLP = []string{
		domain.MustNewAddress("tcp", "localhost", 4317).String(),
	}

	if config.Spec.Ingester.Status.Env == nil {
		config.Spec.Ingester.Status.Env = make(map[string]string)
	}
	config.Spec.Ingester.Status.Env["SIGNOZ_OTEL_COLLECTOR_TIMEOUT"] = "10m"

	return nil
}

func (e *systemdMoldingEnricher) enrichMCP(config *installation.Casting) error {
	if !config.Spec.MCP.Spec.IsEnabled() {
		return nil
	}
	config.Spec.MCP.Status.Addresses.HTTP = []string{
		domain.MustNewAddress("http", "localhost", 8000).String(),
	}
	return nil
}
