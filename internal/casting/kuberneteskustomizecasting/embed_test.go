package kuberneteskustomizecasting

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotEmptyAndValid(t *testing.T) {
	templates := map[string]*domain.Template{
		"clickhouseOperatorClusterrole":        clickhouseOperatorClusterrole,
		"clickhouseOperatorClusterrolebinding": clickhouseOperatorClusterrolebinding,
		"clickhouseOperatorConfigmap":          clickhouseOperatorConfigmap,
		"clickhouseOperatorDeployment":         clickhouseOperatorDeployment,
		"clickhouseOperatorService":            clickhouseOperatorService,
		"clickhouseOperatorServiceaccount":     clickhouseOperatorServiceaccount,
		"clickhouseOperatorKustomization":      clickhouseOperatorKustomization,
		"clickhouseOperatorNamespace":          clickhouseOperatorNamespace,
		"clickhouseInstanceInstallation":       clickhouseInstanceInstallation,
		"clickhouseInstanceConfigmap":          clickhouseInstanceConfigmap,
		"clickhouseInstallationKustomization":  clickhouseInstallationKustomization,
		"clickhouseKeeperInstallation":         clickhouseKeeperInstallation,
		"clickhouseKeeperKustomization":        clickhouseKeeperKustomization,
		"signozService":                        signozService,
		"signozServiceaccount":                 signozServiceaccount,
		"signozStatefulset":                    signozStatefulset,
		"signozKustomization":                  signozKustomization,
		"mcpDeployment":                        mcpDeployment,
		"mcpService":                           mcpService,
		"mcpKustomization":                     mcpKustomization,
		"ingesterConfigmap":                    ingesterConfigmap,
		"ingesterDeployment":                   ingesterDeployment,
		"ingesterService":                      ingesterService,
		"ingesterServiceaccount":               ingesterServiceaccount,
		"ingesterKustomization":                ingesterKustomization,
		"metastoreService":                     metastoreService,
		"metastoreServiceaccount":              metastoreServiceaccount,
		"metastoreStatefulset":                 metastoreStatefulset,
		"metastoreKustomization":               metastoreKustomization,
		"telemetrystoreMigratorJob":            telemetrystoreMigratorJob,
		"telemetrystoreMigratorKustomization":  telemetrystoreMigratorKustomization,
		"deploymentNamespace":                  deploymentNamespace,
		"deploymentKustomization":              deploymentKustomization,
		"telemetryStoreOverrideTemplate":       telemetryStoreOverrideTemplate,
	}

	// The clickhouse templates index Spec.Config.Data, which the default casting types as an empty map; nil cannot be indexed.
	cfg := installation.Default(&installation.Casting{})

	for name, tmpl := range templates {
		assert.NotEmpty(t, tmpl, "%s should not be empty", name)
		buf := bytes.NewBuffer(nil)
		err := tmpl.Execute(buf, cfg)
		assert.NoError(t, err, "error executing %s", name)
		assert.NotEmpty(t, buf.String(), "%s output should not be empty", name)
	}
}

func TestForge(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(cfg *installation.Casting)
		check  func(t *testing.T, materials map[string]domain.StructuredMaterial)
		pass   bool
	}{
		{
			name:   "Default_Valid",
			mutate: func(cfg *installation.Casting) {},
			check: func(t *testing.T, materials map[string]domain.StructuredMaterial) {
				t.Helper()

				assert.Contains(t, materials, "deployment/metastore/postgres/statefulset.yaml")
				assert.NotContains(t, materials, "deployment/mcp/deployment.yaml")
				for _, path := range []string{
					"deployment/signoz/serviceaccount.yaml",
					"deployment/ingester/serviceaccount.yaml",
					"deployment/metastore/postgres/serviceaccount.yaml",
					"deployment/operators/clickhouse-operator/serviceaccount.yaml",
				} {
					assert.Contains(t, materials, path)
				}

				replicas, err := materials["deployment/telemetrystore/clickhouse/clickhouseinstallation.yaml"].GetBytes("spec.configuration.clusters.0.layout.replicasCount")
				require.NoError(t, err)
				assert.Equal(t, "1", string(replicas))

				signoz := materials["deployment/signoz/statefulset.yaml"]
				sa, err := signoz.GetBytes("spec.template.spec.serviceAccountName")
				require.NoError(t, err)
				assert.Equal(t, "signoz-signoz", string(sa))
				claims, _ := signoz.GetBytes("spec.volumeClaimTemplates")
				assert.Empty(t, claims)

				resources, err := materials["deployment/kustomization.yaml"].GetStringSlice("resources")
				require.NoError(t, err)
				assert.Contains(t, resources, "metastore/postgres")
				assert.NotContains(t, resources, "mcp")
				assert.NotContains(t, resources, "operators/clickhouse-operator")

				operator, err := materials["deployment/operators/clickhouse-operator/kustomization.yaml"].GetBytes("namespace")
				require.NoError(t, err)
				assert.Equal(t, "signoz", string(operator))
				assert.Contains(t, materials, "deployment/operators/clickhouse-operator/namespace.yaml")
			},
			pass: true,
		},
		{
			name: "SqliteMetastore_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite
				cfg.Spec.Signoz.Spec.Env = map[string]string{"SIGNOZ_SQLSTORE_SQLITE_PATH": "/var/lib/signoz/signoz.db"}
			},
			check: func(t *testing.T, materials map[string]domain.StructuredMaterial) {
				t.Helper()

				for path := range materials {
					assert.False(t, strings.HasPrefix(path, "deployment/metastore/"), "unexpected material %s", path)
				}

				resources, err := materials["deployment/kustomization.yaml"].GetStringSlice("resources")
				require.NoError(t, err)
				assert.NotContains(t, resources, "metastore/postgres")

				signoz := materials["deployment/signoz/statefulset.yaml"]
				claim, err := signoz.GetBytes("spec.volumeClaimTemplates.0.metadata.name")
				require.NoError(t, err)
				assert.Equal(t, "data", string(claim))
				mount, err := signoz.GetBytes("spec.template.spec.containers.0.volumeMounts.0.mountPath")
				require.NoError(t, err)
				assert.Equal(t, "/var/lib/signoz", string(mount))
			},
			pass: true,
		},
		{
			name: "SignozDisabled_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.Signoz.Spec.Enabled = v1alpha1.BoolPtr(false)
			},
			check: func(t *testing.T, materials map[string]domain.StructuredMaterial) {
				t.Helper()

				assert.NotContains(t, materials, "deployment/signoz/statefulset.yaml")

				resources, err := materials["deployment/kustomization.yaml"].GetStringSlice("resources")
				require.NoError(t, err)
				assert.NotContains(t, resources, "signoz")
			},
			pass: true,
		},
		{
			name: "BooleanEnv_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.Signoz.Spec.Env = map[string]string{"SIGNOZ_USER_ROOT_ENABLED": "true"}
				cfg.Spec.Ingester.Spec.Env = map[string]string{"OTEL_PORT": "4317"}
			},
			check: func(t *testing.T, materials map[string]domain.StructuredMaterial) {
				t.Helper()

				// A quoted scalar survives the YAML->JSON conversion as a JSON string.
				value, err := materials["deployment/signoz/statefulset.yaml"].GetBytes(`spec.template.spec.containers.0.env.#(name=="SIGNOZ_USER_ROOT_ENABLED").value`)
				require.NoError(t, err)
				assert.Equal(t, "true", string(value))
				assert.Contains(t, string(materials["deployment/signoz/statefulset.yaml"].JSONContents()), `"value":"true"`)

				assert.Contains(t, string(materials["deployment/ingester/deployment.yaml"].JSONContents()), `"value":"4317"`)
			},
			pass: true,
		},
		{
			name: "SystemLogOwnership_Valid",
			mutate: func(cfg *installation.Casting) {
				cfg.Spec.TelemetryStore.Spec.Config.Data = map[string]string{
					"config-0-0.yaml": "query_log:\n  partition_by: toYYYYMM(event_date)\n  ttl: event_date + INTERVAL 1 DAY DELETE\n  flush_interval_milliseconds: 30000\npart_log:\n  ttl: event_date + INTERVAL 1 DAY DELETE\ntext_log:\n  ttl: event_date + INTERVAL 1 DAY DELETE\nuser_defined_executable_functions_config: /etc/clickhouse-server/functions/functions.yaml\n",
					"functions.yaml":  "functions:\n  name: histogramQuantile\n",
				}
			},
			check: func(t *testing.T, materials map[string]domain.StructuredMaterial) {
				t.Helper()

				operator := string(materials["deployment/operators/clickhouse-operator/configmap.yaml"].FmtContents())
				assert.Contains(t, operator, "<engine>Engine = MergeTree PARTITION BY toYYYYMM(event_date) ORDER BY event_time TTL event_date + INTERVAL 1 DAY DELETE</engine>")
				assert.Contains(t, operator, "<flush_interval_milliseconds>30000</flush_interval_milliseconds>")
				assert.Contains(t, operator, "01-clickhouse-16-processors_profile_log.xml")
				assert.Contains(t, operator, "TTL event_date + INTERVAL 7 DAY DELETE</engine>")

				chi := string(materials["deployment/telemetrystore/clickhouse/clickhouseinstallation.yaml"].FmtContents())
				assert.NotContains(t, chi, "query_log")
				assert.NotContains(t, chi, "part_log")
				assert.Contains(t, chi, "text_log")

				// ClickHouse picks the UDF parser by extension, so the YAML body ships under a .yaml key and the config points at it.
				assert.Contains(t, chi, "user_defined_executable_functions_config: /etc/clickhouse-server/functions/functions.yaml")
				functions, err := materials["deployment/telemetrystore/clickhouse/configmap.yaml"].GetBytes("data.functions\\.yaml")
				require.NoError(t, err)
				assert.Contains(t, string(functions), "histogramQuantile")
			},
			pass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := installation.Default(&installation.Casting{})
			tc.mutate(cfg)

			materials, err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *cfg, "")
			if !tc.pass {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			byPath := make(map[string]domain.StructuredMaterial, len(materials))
			for _, m := range materials {
				sm, ok := m.(domain.StructuredMaterial)
				require.True(t, ok, "material %s is not structured", m.Path())
				byPath[m.Path()] = sm
			}

			tc.check(t, byPath)
		})
	}
}
