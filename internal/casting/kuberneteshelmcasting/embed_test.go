package kuberneteshelmcasting

import (
	"bytes"
	"context"
	"log/slog"
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
					"telemetryStoreMigrator.timeout",
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

				// The chart's migrator env is a closed set; the timeout is its one knob.
				assert.Equal(t, "10m", get(t, values, "telemetryStoreMigrator.timeout"))
			},
			pass: true,
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

func TestChartSource(t *testing.T) {
	testCases := []struct {
		name            string
		annotations     map[string]string
		expectedChart   string
		expectedVersion string
		expectedRepoURL string
	}{
		{
			name:            "Defaults_Valid",
			expectedChart:   "signoz",
			expectedRepoURL: "https://charts.signoz.io",
		},
		{
			name: "ChartAnnotations_Valid",
			annotations: map[string]string{
				installation.HelmChart.Key:        "signoz",
				installation.HelmChartRepoURL.Key: "https://charts.example.com",
				installation.HelmChartVersion.Key: "0.140.0",
			},
			expectedChart:   "signoz",
			expectedVersion: "0.140.0",
			expectedRepoURL: "https://charts.example.com",
		},
		{
			name:          "LocalPath_NoRepo",
			annotations:   map[string]string{installation.HelmChart.Key: "/charts/signoz"},
			expectedChart: "/charts/signoz",
		},
		{
			name:          "ArchiveURL_NoRepo",
			annotations:   map[string]string{installation.HelmChart.Key: "https://charts.example.com/signoz-0.140.0.tgz"},
			expectedChart: "https://charts.example.com/signoz-0.140.0.tgz",
		},
		{
			name:          "RepoQualified_NoRepo",
			annotations:   map[string]string{installation.HelmChart.Key: "signoz/signoz"},
			expectedChart: "signoz/signoz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := zookeeperCasting()
			cfg.Metadata.Annotations = tc.annotations

			chart, version, repoURL := chartSource(*cfg)

			assert.Equal(t, tc.expectedChart, chart)
			assert.Equal(t, tc.expectedVersion, version)
			assert.Equal(t, tc.expectedRepoURL, repoURL)
		})
	}
}
