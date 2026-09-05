package kuberneteshelmcasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The chart ships ZooKeeper only, so every valid helm casting states the kind.
func zookeeperCasting() *installation.Casting {
	return installation.Default(&installation.Casting{
		Spec: installation.Spec{
			TelemetryKeeper: installation.TelemetryKeeper{Kind: installation.TelemetryKeeperKindZookeeper},
		},
	})
}

func get(t *testing.T, material domain.StructuredMaterial, path string) string {
	t.Helper()

	value, err := material.GetBytes(path)
	require.NoError(t, err, path)

	return string(value)
}

func absent(t *testing.T, material domain.StructuredMaterial, path string) {
	t.Helper()

	value, _ := material.GetBytes(path)
	assert.Empty(t, string(value), "%s should not be rendered", path)
}

func TestNotEmptyAndValid(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	require.NoError(t, valuesYAMLTemplate.Execute(buf, zookeeperCasting()))
	assert.NotEmpty(t, buf.String())
}

func TestForge(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(cfg *installation.Casting)
		check  func(t *testing.T, values domain.StructuredMaterial)
		pass   bool
	}{
		{
			name:   "Zookeeper_Valid",
			mutate: func(cfg *installation.Casting) {},
			check: func(t *testing.T, values domain.StructuredMaterial) {
				t.Helper()

				assert.Equal(t, "signoz", get(t, values, "fullnameOverride"))
				assert.Equal(t, "signoz-telemetrystore-clickhouse", get(t, values, "clickhouse.fullnameOverride"))
				assert.Equal(t, "signoz-telemetrykeeper-zookeeper", get(t, values, "clickhouse.zookeeper.fullnameOverride"))
				assert.Equal(t, "signoz-metastore-postgres", get(t, values, "postgresql.fullnameOverride"))
				assert.Equal(t, "signoz-telemetrystore-migrator", get(t, values, "telemetryStoreMigrator.name"))
				assert.Equal(t, "ingester", get(t, values, "otelCollector.name"))

				// The store's replicas are 0-based in foundry; the chart counts totals.
				assert.Equal(t, "1", get(t, values, "clickhouse.layout.replicasCount"))
				assert.Equal(t, "1", get(t, values, "clickhouse.layout.shardsCount"))
				assert.Equal(t, "1", get(t, values, "clickhouse.zookeeper.replicaCount"))
				assert.Equal(t, "true", get(t, values, "clickhouse.zookeeper.enabled"))
				assert.Equal(t, "true", get(t, values, "postgresql.enabled"))
				assert.Equal(t, "25.12.5", get(t, values, "clickhouse.image.tag"))
				assert.Equal(t, "latest", get(t, values, "signoz.image.tag"))

				for _, path := range []string{
					"clickhouse.enabled",
					"clickhouse.image.registry",
					"signoz.name",
					"signoz.service",
					"otelCollector.enabled",
					"otelCollector.ports",
					"otelCollector.env",
					"telemetryStoreMigrator.enabled",
					"telemetryStoreMigrator.env",
				} {
					absent(t, values, path)
				}
			},
			pass: true,
		},
		{
			name: "StoreReplicas_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.TelemetryStore.Spec.Cluster.Replicas = v1alpha1.IntPtr(2)
				cfg.Spec.TelemetryStore.Spec.Cluster.Shards = v1alpha1.IntPtr(2)
			},
			check: func(t *testing.T, values domain.StructuredMaterial) {
				t.Helper()

				assert.Equal(t, "3", get(t, values, "clickhouse.layout.replicasCount"))
				assert.Equal(t, "2", get(t, values, "clickhouse.layout.shardsCount"))
			},
			pass: true,
		},
		{
			name: "KeeperDisabled_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.TelemetryKeeper.Spec.Enabled = v1alpha1.BoolPtr(false)
			},
			check: func(t *testing.T, values domain.StructuredMaterial) {
				t.Helper()

				assert.Equal(t, "false", get(t, values, "clickhouse.zookeeper.enabled"))
			},
			pass: true,
		},
		{
			name: "Sqlite_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite
				cfg.Spec.Signoz.Spec.Env = map[string]string{"SIGNOZ_SQLSTORE_PROVIDER": "sqlite"}
			},
			check: func(t *testing.T, values domain.StructuredMaterial) {
				t.Helper()

				assert.Equal(t, "false", get(t, values, "postgresql.enabled"))
				absent(t, values, "postgresql.auth")
				absent(t, values, "postgresql.fullnameOverride")
				assert.Equal(t, "sqlite", get(t, values, "signoz.env.SIGNOZ_SQLSTORE_PROVIDER"))
			},
			pass: true,
		},
		{
			name: "PostgresAuth_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.MetaStore.Spec.Env = map[string]string{
					"POSTGRES_DB":       "signoz",
					"POSTGRES_USER":     "signoz",
					"POSTGRES_PASSWORD": "s3cret",
				}
			},
			check: func(t *testing.T, values domain.StructuredMaterial) {
				t.Helper()

				assert.Equal(t, "signoz", get(t, values, "postgresql.auth.database"))
				assert.Equal(t, "signoz", get(t, values, "postgresql.auth.username"))
				assert.Equal(t, "s3cret", get(t, values, "postgresql.auth.password"))
			},
			pass: true,
		},
		{
			name: "IngesterEnv_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.Ingester.Spec.Env = map[string]string{"SIGNOZ_OTEL_COLLECTOR_TIMEOUT": "10m", "FLAG": "false"}
			},
			check: func(t *testing.T, values domain.StructuredMaterial) {
				t.Helper()

				assert.Equal(t, "10m", get(t, values, "otelCollector.additionalEnvs.SIGNOZ_OTEL_COLLECTOR_TIMEOUT"))
				assert.Contains(t, string(values.JSONContents()), `"FLAG":"false"`)
				absent(t, values, "otelCollector.env")
			},
			pass: true,
		},
		{
			name: "RegistryImage_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.Signoz.Spec.Image = "ghcr.io/signoz/signoz:v1"
				cfg.Spec.Ingester.Spec.Image = "localhost:5000/collector:v2"
				cfg.Spec.MetaStore.Spec.Image = "postgres:16"
			},
			check: func(t *testing.T, values domain.StructuredMaterial) {
				t.Helper()

				assert.Equal(t, "ghcr.io", get(t, values, "signoz.image.registry"))
				assert.Equal(t, "signoz/signoz", get(t, values, "signoz.image.repository"))
				assert.Equal(t, "v1", get(t, values, "signoz.image.tag"))

				assert.Equal(t, "localhost:5000", get(t, values, "otelCollector.image.registry"))
				assert.Equal(t, "collector", get(t, values, "otelCollector.image.repository"))
				assert.Equal(t, "v2", get(t, values, "otelCollector.image.tag"))

				absent(t, values, "postgresql.image.registry")
				assert.Equal(t, "library/postgres", get(t, values, "postgresql.image.repository"))
				assert.Equal(t, "16", get(t, values, "postgresql.image.tag"))
			},
			pass: true,
		},
		{
			name: "EmptyImage_Invalid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.Signoz.Spec.Image = ""
			},
			check: func(t *testing.T, values domain.StructuredMaterial) {},
			pass:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := zookeeperCasting()
			tc.mutate(cfg)

			materials, err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *cfg, "")
			if !tc.pass {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)
			require.Len(t, materials, 1)
			assert.Equal(t, "deployment/values.yaml", materials[0].Path())

			values, ok := materials[0].(domain.StructuredMaterial)
			require.True(t, ok)

			tc.check(t, values)
		})
	}
}

func TestEnrichStatus(t *testing.T) {
	testCases := []struct {
		name   string
		cfg    func() *installation.Casting
		kind   v1alpha1.MoldingKind
		check  func(t *testing.T, cfg *installation.Casting)
		pass   bool
		reason string
	}{
		{
			name: "Store_Valid",
			cfg:  zookeeperCasting,
			kind: v1alpha1.MoldingKindTelemetryStore,
			check: func(t *testing.T, cfg *installation.Casting) {
				t.Helper()
				assert.Equal(t, []string{"tcp://signoz-telemetrystore-clickhouse:9000"}, cfg.Spec.TelemetryStore.Status.Addresses.TCP)
			},
			pass: true,
		},
		{
			name: "StoreDisabled_Invalid",
			cfg: func() *installation.Casting {
				cfg := zookeeperCasting()
				cfg.Spec.TelemetryStore.Spec.Enabled = v1alpha1.BoolPtr(false)
				return cfg
			},
			kind:   v1alpha1.MoldingKindTelemetryStore,
			reason: "telemetrystore.spec.enabled: false",
		},
		{
			name: "Zookeeper_Valid",
			cfg:  zookeeperCasting,
			kind: v1alpha1.MoldingKindTelemetryKeeper,
			check: func(t *testing.T, cfg *installation.Casting) {
				t.Helper()
				assert.Equal(t, []string{"tcp://signoz-telemetrykeeper-zookeeper-0:2181"}, cfg.Spec.TelemetryKeeper.Status.Addresses.Client)
				assert.Equal(t, []string{"tcp://signoz-telemetrykeeper-zookeeper-0:2888"}, cfg.Spec.TelemetryKeeper.Status.Addresses.Raft)
			},
			pass: true,
		},
		{
			name:   "UnstatedKeeperKind_Invalid",
			cfg:    func() *installation.Casting { return installation.Default(&installation.Casting{}) },
			kind:   v1alpha1.MoldingKindTelemetryKeeper,
			reason: "telemetrykeeper.kind: clickhousekeeper",
		},
		{
			name: "KeeperDisabled_Valid",
			cfg: func() *installation.Casting {
				cfg := zookeeperCasting()
				cfg.Spec.TelemetryKeeper.Spec.Enabled = v1alpha1.BoolPtr(false)
				return cfg
			},
			kind:  v1alpha1.MoldingKindTelemetryKeeper,
			check: func(t *testing.T, cfg *installation.Casting) {},
			pass:  true,
		},
		{
			name: "Postgres_Valid",
			cfg:  zookeeperCasting,
			kind: v1alpha1.MoldingKindMetaStore,
			check: func(t *testing.T, cfg *installation.Casting) {
				t.Helper()
				assert.Equal(t, []string{"postgres://signoz-metastore-postgres:5432"}, cfg.Spec.MetaStore.Status.Addresses.DSN)
			},
			pass: true,
		},
		{
			name: "Sqlite_Valid",
			cfg: func() *installation.Casting {
				cfg := zookeeperCasting()
				cfg.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite
				return cfg
			},
			kind: v1alpha1.MoldingKindMetaStore,
			check: func(t *testing.T, cfg *installation.Casting) {
				t.Helper()
				assert.Empty(t, cfg.Spec.MetaStore.Status.Addresses.DSN)
			},
			pass: true,
		},
		{
			name: "Signoz_Valid",
			cfg:  zookeeperCasting,
			kind: v1alpha1.MoldingKindSignoz,
			check: func(t *testing.T, cfg *installation.Casting) {
				t.Helper()
				assert.Equal(t, []string{"tcp://signoz:8080"}, cfg.Spec.Signoz.Status.Addresses.APIServer)
				assert.Equal(t, []string{"ws://signoz:4320"}, cfg.Spec.Signoz.Status.Addresses.Opamp)
			},
			pass: true,
		},
		{
			name: "Ingester_Valid",
			cfg:  zookeeperCasting,
			kind: v1alpha1.MoldingKindIngester,
			check: func(t *testing.T, cfg *installation.Casting) {
				t.Helper()
				assert.Equal(t, []string{"tcp://signoz-ingester:4318", "tcp://signoz-ingester:4317"}, cfg.Spec.Ingester.Status.Addresses.OTLP)
			},
			pass: true,
		},
		{
			name: "IngesterDisabled_Invalid",
			cfg: func() *installation.Casting {
				cfg := zookeeperCasting()
				cfg.Spec.Ingester.Spec.Enabled = v1alpha1.BoolPtr(false)
				return cfg
			},
			kind:   v1alpha1.MoldingKindIngester,
			reason: "ingester.spec.enabled: false",
		},
		{
			name:  "MCPDisabled_Valid",
			cfg:   zookeeperCasting,
			kind:  v1alpha1.MoldingKindMCP,
			check: func(t *testing.T, cfg *installation.Casting) {},
			pass:  true,
		},
		{
			name: "MCP_Invalid",
			cfg: func() *installation.Casting {
				cfg := zookeeperCasting()
				cfg.Spec.MCP.Spec.Enabled = v1alpha1.BoolPtr(true)
				return cfg
			},
			kind:   v1alpha1.MoldingKindMCP,
			reason: "does not support mcp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.cfg()

			enricher, err := New(slog.New(slog.DiscardHandler)).Enricher(context.Background(), cfg)
			require.NoError(t, err)

			err = enricher.EnrichStatus(context.Background(), tc.kind, cfg)
			if !tc.pass {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.reason)

				return
			}
			require.NoError(t, err)

			tc.check(t, cfg)
		})
	}
}

func TestRelease(t *testing.T) {
	testCases := []struct {
		name        string
		annotations map[string]string
		values      string
		check       func(t *testing.T, release any)
		pass        bool
	}{
		{
			name:   "Defaults_Valid",
			values: "signoz: {}\n",
			pass:   true,
		},
		{
			name: "ChartAnnotations_Valid",
			annotations: map[string]string{
				installation.HelmChart.Key:         "mirror/signoz",
				installation.HelmChartRepoURL.Key:  "https://charts.example.com",
				installation.HelmChartRepoName.Key: "mirror",
				installation.HelmChartVersion.Key:  "0.140.0",
			},
			values: "signoz: {}\n",
			pass:   true,
		},
		{
			name: "UnforgedValues_Invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := zookeeperCasting()
			cfg.Metadata.Annotations = tc.annotations

			dir := t.TempDir()
			if tc.values != "" {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "deployment"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment", "values.yaml"), []byte(tc.values), 0o644))
			}

			release, err := New(slog.New(slog.DiscardHandler)).release(*cfg, dir)
			if !tc.pass {
				require.Error(t, err)

				return
			}
			require.NoError(t, err)

			assert.Equal(t, "signoz", release.Name)
			assert.Equal(t, "signoz", release.Namespace)
			assert.NotNil(t, release.Values)
			assert.Equal(t, installation.HelmChart.Resolve(tc.annotations), release.Chart)
			assert.Equal(t, installation.HelmChartVersion.Resolve(tc.annotations), release.Version)
			assert.Equal(t, installation.HelmChartRepoName.Resolve(tc.annotations), release.Repo.Name)
			assert.Equal(t, installation.HelmChartRepoURL.Resolve(tc.annotations), release.Repo.URL)
		})
	}
}
