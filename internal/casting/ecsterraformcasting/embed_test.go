package ecsterraformcasting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"text/template"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/molding/ingestermolding"
	"github.com/signoz/foundry/internal/molding/mcpmolding"
	"github.com/signoz/foundry/internal/molding/metastoremolding"
	"github.com/signoz/foundry/internal/molding/signozmolding"
	"github.com/signoz/foundry/internal/molding/telemetrykeepermolding"
	"github.com/signoz/foundry/internal/molding/telemetrystoremolding"
	"github.com/signoz/foundry/internal/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func statedCasting(declared *installation.Casting) *installation.Casting {
	c := installation.Default(declared)
	c.Metadata.Annotations = map[string]string{
		installation.ECSRegion.Key:           "us-east-1",
		installation.ECSClusterARN.Key:       "arn:aws:ecs:us-east-1:123456789012:cluster/test",
		installation.ECSVPCID.Key:            "vpc-abc123",
		installation.ECSPrivateSubnetIDs.Key: "subnet-abc123, subnet-def456",
		installation.ECSSecurityGroupIDs.Key: "sg-abc123",
	}

	return c
}

func boundCasting(declared *installation.Casting) *installation.Casting {
	c := installation.Default(declared)
	c.Metadata.Annotations = map[string]string{
		installation.ECSRegion.Key: "us-east-1",
	}
	c.Spec.Infrastructure.Name = "foundry"

	return c
}

func templateDataFor(t *testing.T, casting *installation.Casting) templateData {
	t.Helper()

	data, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
	require.NoError(t, err)

	return data
}

func requireTerraform(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping terraform test in short mode")
	}

	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("terraform is not available")
	}
}

// The moldings write the config files the roots read with file(), so a plan
// needs the casting molded as forge molds it.
func moldedCasting(t *testing.T, casting *installation.Casting) *installation.Casting {
	t.Helper()

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	enricher, err := New(logger).Enricher(ctx, casting)
	require.NoError(t, err)

	moldings := []molding.Molding{
		telemetrykeepermolding.New(logger),
		telemetrystoremolding.New(logger),
		metastoremolding.New(logger),
		signozmolding.New(logger),
		ingestermolding.New(logger),
		mcpmolding.New(logger),
	}

	for _, m := range moldings {
		require.NoError(t, enricher.EnrichStatus(ctx, m.Kind(), casting))
	}

	for _, m := range moldings {
		require.NoError(t, m.MoldV1Alpha1(ctx, casting))
	}

	require.NoError(t, casting.MergeStatusIntoSpec())

	return casting
}

// One cache serves every root, and no root carries a lock file to pin it.
func terraform(t *testing.T, root, cache string, args ...string) (string, error) {
	t.Helper()

	out := bytes.NewBuffer(nil)

	cmd := exec.Command("terraform", append([]string{"-chdir=" + root}, args...)...)
	cmd.Env = append(os.Environ(), "TF_PLUGIN_CACHE_DIR="+cache, "TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE=true", "TF_IN_AUTOMATION=1")
	cmd.Stdout = out
	cmd.Stderr = out

	err := cmd.Run()

	return out.String(), err
}

func TestEveryTemplateRendersValidJSON(t *testing.T) {
	data := templateDataFor(t, installation.Default(&installation.Casting{}))

	templates := map[string]*domain.Template{
		"versions.tf.json":        versionsTF,
		"backend.tf.json":         backendTF,
		"providers.tf.json":       providersTF,
		"main.tf.json":            mainTF,
		"variables.tf.json":       variablesTF,
		"outputs.tf.json":         outputsTF,
		"terraform.tfvars.json":   tfarsTF,
		"telemetrykeeper.tf.json": telemetryKeeperTF,
		"telemetrystore.tf.json":  telemetryStoreTF,
		"migrator.tf.json":        migratorTF,
		"metastore.tf.json":       metaStoreTF,
		"signoz.tf.json":          signozTF,
		"ingester.tf.json":        ingesterTF,
		"mcp.tf.json":             mcpTF,
	}

	for name, tmpl := range templates {
		buf := bytes.NewBuffer(nil)
		require.NoError(t, tmpl.Execute(buf, data), name)

		_, err := domain.NewJSONMaterial(buf.Bytes(), name)
		require.NoError(t, err, name)
	}
}

func TestTemplateDataResolution(t *testing.T) {
	complete := statedCasting(&installation.Casting{}).Metadata.Annotations

	malformed := map[string]string{}
	maps.Copy(malformed, complete)
	malformed[installation.ECSPrivateSubnetIDs.Key] = " , ,"

	roles := map[string]string{
		installation.ECSTaskRoleARN.Key:          "arn:aws:iam::123456789012:role/task",
		installation.ECSTaskExecutionRoleARN.Key: "arn:aws:iam::123456789012:role/exec",
	}

	tests := []struct {
		name            string
		annotations     map[string]string
		infrastructure  string
		expectedRegion  string
		expectedStated  map[string]any
		expectedBound   bool
		expectedMessage string
		pass            bool
	}{
		{
			name:           "AllStated_Valid",
			annotations:    complete,
			expectedRegion: "us-east-1",
			expectedStated: map[string]any{
				"ClusterARN":       "arn:aws:ecs:us-east-1:123456789012:cluster/test",
				"VPCID":            "vpc-abc123",
				"SubnetIDs":        []string{"subnet-abc123", "subnet-def456"},
				"SecurityGroupIDs": []string{"sg-abc123"},
			},
			pass: true,
		},
		{name: "NothingStated_Valid", annotations: nil, expectedStated: map[string]any{}, pass: true},
		{name: "SubnetIDsMalformed_Invalid", annotations: malformed, expectedMessage: "no ids found"},
		{name: "Bound_Valid", infrastructure: "foundry", expectedStated: map[string]any{}, expectedBound: true, pass: true},
		{
			name:           "BoundRolesStated_Valid",
			annotations:    roles,
			infrastructure: "foundry",
			expectedStated: map[string]any{
				"TaskRoleARN":      "arn:aws:iam::123456789012:role/task",
				"ExecutionRoleARN": "arn:aws:iam::123456789012:role/exec",
			},
			expectedBound: true,
			pass:          true,
		},
		{name: "BoundNameMalformed_Invalid", infrastructure: "Foundry_", expectedMessage: "failed to resolve the substrate the installation is bound to"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			casting := installation.Default(&installation.Casting{})
			casting.Metadata.Annotations = test.annotations
			casting.Spec.Infrastructure.Name = test.infrastructure

			data, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)

			if test.pass {
				require.NoError(t, err)
				assert.Equal(t, test.expectedRegion, data.Region)
				assert.Equal(t, test.expectedStated, data.Stated)
				assert.Equal(t, test.expectedBound, data.Substrate != nil)

				return
			}

			require.Error(t, err)
			assert.ErrorContains(t, err, test.expectedMessage)
		})
	}
}

// Sqlite is a file the signoz task holds, so it is no service of its own.
func TestForgeSelectsComponents(t *testing.T) {
	sqlite := statedCasting(&installation.Casting{})
	sqlite.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite

	withMCP := statedCasting(&installation.Casting{})
	withMCP.Spec.MCP.Spec.Enabled = v1alpha1.BoolPtr(true)

	tests := []struct {
		name           string
		casting        *installation.Casting
		path           string
		expectedForged bool
	}{
		{name: "Postgres_MetaStoreForged", casting: statedCasting(&installation.Casting{}), path: "metastore.tf.json", expectedForged: true},
		{name: "Sqlite_MetaStoreUnforged", casting: sqlite, path: "metastore.tf.json"},
		{name: "MCPEnabled_Forged", casting: withMCP, path: "mcp.tf.json", expectedForged: true},
		{name: "MCPDisabled_Unforged", casting: statedCasting(&installation.Casting{}), path: "mcp.tf.json"},
		{name: "NothingStated_Forged", casting: installation.Default(&installation.Casting{}), path: "main.tf.json", expectedForged: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			materials, err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *tt.casting, "")
			require.NoError(t, err)

			forged := false
			for _, material := range materials {
				if strings.Contains(material.Path(), tt.path) {
					forged = true
				}
			}

			assert.Equal(t, tt.expectedForged, forged)
		})
	}
}

// A public subnet hands an awsvpc task on the EC2 launch type a route it
// cannot use. An empty "data" object is a root terraform refuses outright.
func TestStatedSubnetsAreCheckedAtPlan(t *testing.T) {
	tests := []struct {
		name        string
		subnetIDs   string
		expectedIDs []string
		pass        bool
	}{
		{name: "TwoSubnetsStated_Checked", subnetIDs: "subnet-abc123, subnet-def456", expectedIDs: []string{"subnet-abc123", "subnet-def456"}, pass: true},
		{name: "NoSubnetStated_NotChecked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := statedCasting(&installation.Casting{})

			if tt.subnetIDs == "" {
				delete(casting.Metadata.Annotations, installation.ECSPrivateSubnetIDs.Key)
			} else {
				casting.Metadata.Annotations[installation.ECSPrivateSubnetIDs.Key] = tt.subnetIDs
			}

			main := bytes.NewBuffer(nil)
			require.NoError(t, mainTF.Execute(main, templateDataFor(t, casting)))

			material, err := domain.NewJSONMaterial(main.Bytes(), "main.tf.json")
			require.NoError(t, err)

			preconditions, err := material.GetBytes("resource.aws_service_discovery_private_dns_namespace.main.lifecycle.precondition")
			if !tt.pass {
				assert.Error(t, err, "an unstated subnet list emits no precondition")

				_, err := material.GetBytes("data")
				assert.Error(t, err, "an unstated subnet list emits no lookup")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, len(tt.expectedIDs), strings.Count(string(preconditions), `"condition"`))

			for _, id := range tt.expectedIDs {
				assert.Contains(t, string(preconditions), fmt.Sprintf("subnet %s assigns public IPs on launch", id))
			}

			lookups, err := material.GetBytes("data.aws_subnet")
			require.NoError(t, err)
			assert.Equal(t, len(tt.expectedIDs), strings.Count(string(lookups), `"id"`))
		})
	}
}

// Terraform refuses an empty or malformed value at plan, so every variable the
// casting does not judge carries its own condition, and a default would race
// the tfvars.
func TestVariablesCarryValidationsAndNoDefault(t *testing.T) {
	casting := statedCasting(&installation.Casting{})
	casting.Metadata.Annotations[installation.ECSTaskRoleARN.Key] = "arn:aws:iam::123456789012:role/task"
	casting.Metadata.Annotations[installation.ECSTaskExecutionRoleARN.Key] = "arn:aws:iam::123456789012:role/exec"

	buf := bytes.NewBuffer(nil)
	require.NoError(t, variablesTF.Execute(buf, templateDataFor(t, casting)))

	assert.NotContains(t, buf.String(), `"default"`)

	material, err := domain.NewJSONMaterial(buf.Bytes(), "variables.tf.json")
	require.NoError(t, err)

	tests := []struct {
		name              string
		variable          string
		expectedCondition string
	}{
		{name: "Region_Validated", variable: "aws_region", expectedCondition: `^[a-z]{2}(-gov)?-[a-z]+-[0-9]$`},
		{name: "ClusterARN_Validated", variable: "cluster_arn", expectedCondition: `^arn:aws:ecs:`},
		{name: "SubnetIDs_Validated", variable: "subnet_ids", expectedCondition: `^subnet-`},
		{name: "SecurityGroupIDs_Validated", variable: "security_group_ids", expectedCondition: `^sg-`},
		{name: "VPCID_Validated", variable: "vpc_id", expectedCondition: `^vpc-`},
		{name: "TaskRoleARN_Validated", variable: "task_role_arn", expectedCondition: `^arn:aws:iam::`},
		{name: "ExecutionRoleARN_Validated", variable: "execution_role_arn", expectedCondition: `^arn:aws:iam::`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition, err := material.GetBytes("variable." + tt.variable + ".validation.condition")
			require.NoError(t, err)
			assert.Contains(t, string(condition), tt.expectedCondition)
		})
	}
}

// No name is computed for a role this stack does not create.
func TestTfvarsCarriesEveryStatedValue(t *testing.T) {
	statedRoles := statedCasting(&installation.Casting{})
	statedRoles.Metadata.Annotations[installation.ECSTaskRoleARN.Key] = "arn:aws:iam::123456789012:role/task"
	statedRoles.Metadata.Annotations[installation.ECSTaskExecutionRoleARN.Key] = "arn:aws:iam::123456789012:role/exec"

	tfvars := `{
		"aws_region": "us-east-1",
		"cluster_arn": "arn:aws:ecs:us-east-1:123456789012:cluster/test",
		"subnet_ids": ["subnet-abc123", "subnet-def456"],
		"security_group_ids": ["sg-abc123"],
		"vpc_id": "vpc-abc123",
		%s
	}`

	tests := []struct {
		name          string
		casting       *installation.Casting
		expectedRoles string
	}{
		{
			name:          "CreatedRoles_Named",
			casting:       statedCasting(&installation.Casting{}),
			expectedRoles: `"task_role_name": "signoz-installation-iam-task", "execution_role_name": "signoz-installation-iam-exec"`,
		},
		{
			name:          "StatedRoles_Carried",
			casting:       statedRoles,
			expectedRoles: `"task_role_arn": "arn:aws:iam::123456789012:role/task", "execution_role_arn": "arn:aws:iam::123456789012:role/exec"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			require.NoError(t, tfarsTF.Execute(buf, templateDataFor(t, tt.casting)))

			assert.JSONEq(t, fmt.Sprintf(tfvars, tt.expectedRoles), buf.String())
		})
	}
}

// A clustered fixture is one service per node, each named by the ordinal the
// molding gives it.
func TestStatefulComponentsForgeAServicePerNode(t *testing.T) {
	shards, perShard, keepers, nodes := 2, 1, 3, 2

	store := statedCasting(&installation.Casting{})
	store.Spec.TelemetryStore.Spec.Cluster.Shards = &shards
	store.Spec.TelemetryStore.Spec.Cluster.Replicas = &perShard

	keeper := statedCasting(&installation.Casting{})
	keeper.Spec.TelemetryKeeper.Kind = installation.TelemetryKeeperKindClickhouseKeeper
	keeper.Spec.TelemetryKeeper.Spec.Cluster.Replicas = &keepers

	metaStore := statedCasting(&installation.Casting{})
	metaStore.Spec.MetaStore.Spec.Cluster.Replicas = &nodes

	signoz := statedCasting(&installation.Casting{})
	signoz.Spec.Signoz.Spec.Cluster.Replicas = &nodes

	tests := []struct {
		name             string
		template         *domain.Template
		casting          *installation.Casting
		expectedServices []string
	}{
		{name: "TelemetryStore_ServicePerNode", template: telemetryStoreTF, casting: store, expectedServices: []string{"signoz-telemetrystore-clickhouse-0-0", "signoz-telemetrystore-clickhouse-0-1", "signoz-telemetrystore-clickhouse-1-0", "signoz-telemetrystore-clickhouse-1-1"}},
		{name: "TelemetryKeeper_ServicePerNode", template: telemetryKeeperTF, casting: keeper, expectedServices: []string{"signoz-telemetrykeeper-clickhousekeeper-0", "signoz-telemetrykeeper-clickhousekeeper-1", "signoz-telemetrykeeper-clickhousekeeper-2"}},
		{name: "MetaStore_ServicePerNode", template: metaStoreTF, casting: metaStore, expectedServices: []string{"signoz-metastore-postgres-0", "signoz-metastore-postgres-1"}},
		{name: "Signoz_ServicePerNode", template: signozTF, casting: signoz, expectedServices: []string{"signoz-signoz-0", "signoz-signoz-1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			require.NoError(t, tt.template.Execute(buf, templateDataFor(t, tt.casting)))

			material, err := domain.NewJSONMaterial(buf.Bytes(), "component.tf.json")
			require.NoError(t, err)

			services, err := material.GetStringSlice("resource.aws_ecs_service.@values.#.name")
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expectedServices, services)
		})
	}
}

// ZooKeeper numbers its own nodes from one, and an ensemble names itself
// 0.0.0.0 so the node does not dial its own service record.
func TestZookeeperEnsembleKnowsItsPeers(t *testing.T) {
	tests := []struct {
		name            string
		replicas        int
		node            int
		expectedID      string
		expectedServers string
	}{
		{name: "SingleNode_NoEnsemble", replicas: 1, node: 0, expectedID: "1"},
		{
			name: "FirstOfThree_Ensemble", replicas: 3, node: 0, expectedID: "1",
			expectedServers: "0.0.0.0:2888:3888,telemetrykeeper-zookeeper-1.signoz-installation.local:2888:3888,telemetrykeeper-zookeeper-2.signoz-installation.local:2888:3888",
		},
		{
			name: "LastOfThree_Ensemble", replicas: 3, node: 2, expectedID: "3",
			expectedServers: "telemetrykeeper-zookeeper-0.signoz-installation.local:2888:3888,telemetrykeeper-zookeeper-1.signoz-installation.local:2888:3888,0.0.0.0:2888:3888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := statedCasting(&installation.Casting{})
			casting.Spec.TelemetryKeeper.Kind = installation.TelemetryKeeperKindZookeeper
			casting.Spec.TelemetryKeeper.Spec.Cluster.Replicas = &tt.replicas

			buf := bytes.NewBuffer(nil)
			require.NoError(t, telemetryKeeperTF.Execute(buf, templateDataFor(t, casting)))

			material, err := domain.NewJSONMaterial(buf.Bytes(), "telemetrykeeper.tf.json")
			require.NoError(t, err)

			container := fmt.Sprintf(`locals.containers_telemetrykeeper_zookeeper_%d.#(name=="signoz-telemetrykeeper-zookeeper-%d")`, tt.node, tt.node)

			id, err := material.GetBytes(container + `.environment.#(name=="ZOO_SERVER_ID").value`)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedID, string(id))

			servers, err := material.GetBytes(container + `.environment.#(name=="ZOO_SERVERS").value`)
			if tt.expectedServers == "" {
				assert.Error(t, err, "a single node stands alone")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedServers, string(servers))
		})
	}
}

// A patch written against the singular output keeps working, so both shapes
// stand.
func TestOutputsEnumerateEveryNode(t *testing.T) {
	shards, perShard, keepers := 2, 1, 3

	casting := statedCasting(&installation.Casting{})
	casting.Spec.TelemetryStore.Spec.Cluster.Shards = &shards
	casting.Spec.TelemetryStore.Spec.Cluster.Replicas = &perShard
	casting.Spec.TelemetryKeeper.Spec.Cluster.Replicas = &keepers

	buf := bytes.NewBuffer(nil)
	require.NoError(t, outputsTF.Execute(buf, templateDataFor(t, casting)))

	material, err := domain.NewJSONMaterial(buf.Bytes(), "outputs.tf.json")
	require.NoError(t, err)

	for path, expectedCount := range map[string]int{
		"output.telemetrystore_service_names.value":  4,
		"output.telemetrykeeper_service_names.value": 3,
	} {
		names, err := material.GetStringSlice(path)
		require.NoError(t, err, "reading %s", path)
		assert.Len(t, names, expectedCount, path)
	}

	for _, path := range []string{"output.metastore_service_name.value", "output.signoz_service_name.value", "output.signoz_service_arn.value"} {
		_, err := material.GetBytes(path)
		assert.NoError(t, err, "reading %s", path)
	}
}

func TestBindingDerivesEveryClusterObject(t *testing.T) {
	boundAndStated := boundCasting(&installation.Casting{})
	boundAndStated.Metadata.Annotations[installation.ECSVPCID.Key] = "vpc-abc123"

	lookups := map[string]string{
		"cluster_arn":        "aws_ecs_cluster",
		"vpc_id":             "aws_vpc",
		"subnet_ids":         "aws_subnets",
		"security_group_ids": "aws_security_group",
	}

	tests := []struct {
		name            string
		casting         *installation.Casting
		expectedBound   bool
		expectedLocals  map[string]string
		expectedMessage string
		pass            bool
	}{
		{
			name:          "Bound_Valid",
			casting:       boundCasting(&installation.Casting{}),
			expectedBound: true,
			expectedLocals: map[string]string{
				"cluster_arn":        "${data.aws_ecs_cluster.substrate.arn}",
				"vpc_id":             "${data.aws_vpc.substrate.id}",
				"subnet_ids":         "${data.aws_subnets.substrate.ids}",
				"security_group_ids": "${data.aws_security_group.substrate.id}",
			},
			pass: true,
		},
		{
			name:    "Unbound_Valid",
			casting: installation.Default(&installation.Casting{}),
			expectedLocals: map[string]string{
				"cluster_arn":        "${var.cluster_arn}",
				"vpc_id":             "${var.vpc_id}",
				"subnet_ids":         "${var.subnet_ids}",
				"security_group_ids": "${var.security_group_ids}",
			},
			pass: true,
		},
		{
			name:            "BoundAndStated_Invalid",
			casting:         boundAndStated,
			expectedMessage: `states the "foundry.signoz.io/ecs-vpc-id" annotation`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := New(slog.New(slog.DiscardHandler)).templateData(*test.casting)

			if test.pass {
				require.NoError(t, err)

				main := bytes.NewBuffer(nil)
				require.NoError(t, mainTF.Execute(main, data))

				mainMaterial, err := domain.NewJSONMaterial(main.Bytes(), "main.tf.json")
				require.NoError(t, err)

				variables := bytes.NewBuffer(nil)
				require.NoError(t, variablesTF.Execute(variables, data))

				variablesMaterial, err := domain.NewJSONMaterial(variables.Bytes(), "variables.tf.json")
				require.NoError(t, err)

				tfvars := bytes.NewBuffer(nil)
				require.NoError(t, tfarsTF.Execute(tfvars, data))

				tfvarsMaterial, err := domain.NewJSONMaterial(tfvars.Bytes(), "terraform.tfvars.json")
				require.NoError(t, err)

				if !test.expectedBound {
					_, err := mainMaterial.GetBytes("data")
					assert.Error(t, err, "an unbound casting stating nothing looks nothing up")
				}

				for variable, dataSource := range lookups {
					local, err := mainMaterial.GetBytes("locals." + variable)
					require.NoError(t, err, variable)
					assert.Contains(t, string(local), test.expectedLocals[variable], variable)

					_, lookupErr := mainMaterial.GetBytes("data." + dataSource + ".substrate")
					_, declaredErr := variablesMaterial.GetBytes("variable." + variable)
					_, carriedErr := tfvarsMaterial.GetBytes(variable)

					assert.Equal(t, test.expectedBound, lookupErr == nil, "%s is looked up", variable)
					assert.Equal(t, !test.expectedBound, declaredErr == nil, "%s is declared", variable)
					assert.Equal(t, !test.expectedBound, carriedErr == nil, "%s is carried", variable)
				}

				return
			}

			require.Error(t, err)
			assert.ErrorContains(t, err, test.expectedMessage)
			assert.Equal(t, foundryerrors.TypeInvalidInput.ExitCode(), foundryerrors.ExitCode(err))
		})
	}
}

// A task with no node of its storage class stays pending, so a casting bound
// to nothing places nothing.
func TestBoundServicesArePlacedByStorageClass(t *testing.T) {
	sqlite := boundCasting(&installation.Casting{})
	sqlite.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite

	zookeeper := boundCasting(&installation.Casting{})
	zookeeper.Spec.TelemetryKeeper.Kind = installation.TelemetryKeeperKindZookeeper

	cluster := boundCasting(&installation.Casting{})
	cluster.Spec.TelemetryStore.Spec.Cluster.Shards = v1alpha1.IntPtr(2)
	cluster.Spec.TelemetryStore.Spec.Cluster.Replicas = v1alpha1.IntPtr(1)
	cluster.Spec.TelemetryKeeper.Spec.Cluster.Replicas = v1alpha1.IntPtr(3)
	cluster.Spec.MetaStore.Spec.Cluster.Replicas = v1alpha1.IntPtr(2)

	sqliteCluster := boundCasting(&installation.Casting{})
	sqliteCluster.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite
	sqliteCluster.Spec.TelemetryKeeper.Kind = installation.TelemetryKeeperKindZookeeper
	sqliteCluster.Spec.TelemetryKeeper.Spec.Cluster.Replicas = v1alpha1.IntPtr(3)
	sqliteCluster.Spec.Signoz.Spec.Cluster.Replicas = v1alpha1.IntPtr(2)

	tests := []struct {
		name          string
		template      *domain.Template
		casting       *installation.Casting
		expectedClass string
		pass          bool
	}{
		{name: "TelemetryStoreOnPostgres_Persistent", template: telemetryStoreTF, casting: boundCasting(&installation.Casting{}), expectedClass: "persistent", pass: true},
		{name: "TelemetryStoreOnSqlite_Persistent", template: telemetryStoreTF, casting: sqlite, expectedClass: "persistent", pass: true},
		{name: "TelemetryKeeperOnPostgres_Persistent", template: telemetryKeeperTF, casting: boundCasting(&installation.Casting{}), expectedClass: "persistent", pass: true},
		{name: "TelemetryKeeperOnSqlite_Persistent", template: telemetryKeeperTF, casting: sqlite, expectedClass: "persistent", pass: true},
		{name: "Zookeeper_Persistent", template: telemetryKeeperTF, casting: zookeeper, expectedClass: "persistent", pass: true},
		{name: "MetaStoreOnPostgres_Persistent", template: metaStoreTF, casting: boundCasting(&installation.Casting{}), expectedClass: "persistent", pass: true},
		{name: "SignozOnPostgres_Ephemeral", template: signozTF, casting: boundCasting(&installation.Casting{}), expectedClass: "ephemeral", pass: true},
		{name: "SignozOnSqlite_Persistent", template: signozTF, casting: sqlite, expectedClass: "persistent", pass: true},
		{name: "IngesterOnPostgres_Ephemeral", template: ingesterTF, casting: boundCasting(&installation.Casting{}), expectedClass: "ephemeral", pass: true},
		{name: "IngesterOnSqlite_Ephemeral", template: ingesterTF, casting: sqlite, expectedClass: "ephemeral", pass: true},
		{name: "MCPOnPostgres_Ephemeral", template: mcpTF, casting: boundCasting(&installation.Casting{}), expectedClass: "ephemeral", pass: true},
		{name: "MCPOnSqlite_Ephemeral", template: mcpTF, casting: sqlite, expectedClass: "ephemeral", pass: true},
		{name: "TelemetryStoreCluster_Persistent", template: telemetryStoreTF, casting: cluster, expectedClass: "persistent", pass: true},
		{name: "TelemetryKeeperCluster_Persistent", template: telemetryKeeperTF, casting: cluster, expectedClass: "persistent", pass: true},
		{name: "ZookeeperCluster_Persistent", template: telemetryKeeperTF, casting: sqliteCluster, expectedClass: "persistent", pass: true},
		{name: "MetaStoreCluster_Persistent", template: metaStoreTF, casting: cluster, expectedClass: "persistent", pass: true},
		{name: "SignozClusterOnPostgres_Ephemeral", template: signozTF, casting: cluster, expectedClass: "ephemeral", pass: true},
		{name: "SignozClusterOnSqlite_Persistent", template: signozTF, casting: sqliteCluster, expectedClass: "persistent", pass: true},
		{name: "TelemetryStoreUnbound_Valid", template: telemetryStoreTF, casting: statedCasting(&installation.Casting{}), pass: true},
		{name: "TelemetryKeeperUnbound_Valid", template: telemetryKeeperTF, casting: statedCasting(&installation.Casting{}), pass: true},
		{name: "MetaStoreUnbound_Valid", template: metaStoreTF, casting: statedCasting(&installation.Casting{}), pass: true},
		{name: "SignozUnbound_Valid", template: signozTF, casting: statedCasting(&installation.Casting{}), pass: true},
		{name: "IngesterUnbound_Valid", template: ingesterTF, casting: statedCasting(&installation.Casting{}), pass: true},
		{name: "MCPUnbound_Valid", template: mcpTF, casting: statedCasting(&installation.Casting{}), pass: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := templateDataFor(t, test.casting)

			buf := bytes.NewBuffer(nil)
			err := test.template.Execute(buf, data)

			if test.pass {
				require.NoError(t, err)

				if test.expectedClass == "" {
					assert.NotContains(t, buf.String(), "placement_constraints")
					assert.NotContains(t, buf.String(), "local.seats")
					assert.NotContains(t, buf.String(), "aws_ec2_tag.claims")

					return
				}

				material, err := domain.NewJSONMaterial(buf.Bytes(), "component.tf.json")
				require.NoError(t, err)

				services, err := material.GetStringSlice("resource.aws_ecs_service.@values.#.name")
				require.NoError(t, err)

				types, err := material.GetStringSlice("resource.aws_ecs_service.@values.#.placement_constraints.0.type")
				require.NoError(t, err)

				expressions, err := material.GetStringSlice("resource.aws_ecs_service.@values.#.placement_constraints.0.expression")
				require.NoError(t, err)

				require.Len(t, expressions, len(services), "every service is placed")

				for i, service := range services {
					expectedExpression := "attribute:foundry.signoz.io/storage == " + test.expectedClass

					var expectedDependencies []string

					identity := strings.TrimPrefix(service, test.casting.Metadata.Name+"-")
					if test.expectedClass == "persistent" {
						assert.Contains(t, data.Substrate["Identities"], identity, "%s has a seat", service)
						expectedExpression = fmt.Sprintf("ec2InstanceId == '${local.seats[%q]}' and %s", identity, expectedExpression)
						expectedDependencies = []string{"aws_ec2_tag.claims"}
					}

					assert.Equal(t, "memberOf", types[i])
					assert.Equal(t, expectedExpression, expressions[i], service)

					dependencies, _ := material.GetStringSlice(fmt.Sprintf("resource.aws_ecs_service.@values.%d.depends_on", i))
					assert.Equal(t, expectedDependencies, dependencies, "%s waits for its claim", service)
				}

				return
			}

			require.Error(t, err)
		})
	}
}

func TestBindingClaimsEveryPersistentIdentity(t *testing.T) {
	sqlite := boundCasting(&installation.Casting{})
	sqlite.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite

	cluster := boundCasting(&installation.Casting{})
	cluster.Spec.TelemetryStore.Spec.Cluster.Shards = v1alpha1.IntPtr(2)
	cluster.Spec.TelemetryStore.Spec.Cluster.Replicas = v1alpha1.IntPtr(1)
	cluster.Spec.TelemetryKeeper.Spec.Cluster.Replicas = v1alpha1.IntPtr(3)
	cluster.Spec.MetaStore.Spec.Cluster.Replicas = v1alpha1.IntPtr(2)

	sqliteCluster := boundCasting(&installation.Casting{})
	sqliteCluster.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite
	sqliteCluster.Spec.TelemetryKeeper.Kind = installation.TelemetryKeeperKindZookeeper
	sqliteCluster.Spec.Signoz.Spec.Cluster.Replicas = v1alpha1.IntPtr(2)
	sqliteCluster.Spec.TelemetryStore.Spec.Cluster.Replicas = v1alpha1.IntPtr(1)

	keeperDisabled := boundCasting(&installation.Casting{})
	keeperDisabled.Spec.TelemetryKeeper.Spec.Enabled = v1alpha1.BoolPtr(false)

	malformed := boundCasting(&installation.Casting{})
	malformed.Spec.Infrastructure.Name = "Foundry_"

	tests := []struct {
		name               string
		casting            *installation.Casting
		expectedClaims     bool
		expectedIdentities []string
		pass               bool
	}{
		{
			name:               "Postgres_Valid",
			casting:            boundCasting(&installation.Casting{}),
			expectedClaims:     true,
			expectedIdentities: []string{"telemetrykeeper-clickhousekeeper-0", "metastore-postgres-0", "telemetrystore-clickhouse-0-0"},
			pass:               true,
		},
		{
			name:               "Sqlite_Valid",
			casting:            sqlite,
			expectedClaims:     true,
			expectedIdentities: []string{"telemetrykeeper-clickhousekeeper-0", "signoz-0", "telemetrystore-clickhouse-0-0"},
			pass:               true,
		},
		{
			name:           "PostgresCluster_Valid",
			casting:        cluster,
			expectedClaims: true,
			expectedIdentities: []string{
				"telemetrykeeper-clickhousekeeper-0", "telemetrykeeper-clickhousekeeper-1", "telemetrykeeper-clickhousekeeper-2",
				"metastore-postgres-0", "metastore-postgres-1",
				"telemetrystore-clickhouse-0-0", "telemetrystore-clickhouse-0-1", "telemetrystore-clickhouse-1-0", "telemetrystore-clickhouse-1-1",
			},
			pass: true,
		},
		{
			name:           "SqliteCluster_Valid",
			casting:        sqliteCluster,
			expectedClaims: true,
			expectedIdentities: []string{
				"telemetrykeeper-zookeeper-0",
				"signoz-0", "signoz-1",
				"telemetrystore-clickhouse-0-0", "telemetrystore-clickhouse-0-1",
			},
			pass: true,
		},
		{
			name:               "KeeperDisabled_Valid",
			casting:            keeperDisabled,
			expectedClaims:     true,
			expectedIdentities: []string{"metastore-postgres-0", "telemetrystore-clickhouse-0-0"},
			pass:               true,
		},
		{name: "Unbound_Valid", casting: statedCasting(&installation.Casting{}), pass: true},
		{name: "BoundNameMalformed_Invalid", casting: malformed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := New(slog.New(slog.DiscardHandler)).templateData(*test.casting)

			if test.pass {
				require.NoError(t, err)

				buf := bytes.NewBuffer(nil)
				require.NoError(t, mainTF.Execute(buf, data))

				main, err := domain.NewJSONMaterial(buf.Bytes(), "main.tf.json")
				require.NoError(t, err)

				for _, path := range []string{
					"data.aws_instances.persistent",
					"data.aws_instance.persistent",
					"data.aws_ebs_volumes.persistent",
					"data.aws_ebs_volume.persistent",
					"locals.identities",
					"locals.seats",
					"resource.aws_ec2_tag.claims",
				} {
					_, err := main.GetBytes(path)
					assert.Equal(t, test.expectedClaims, err == nil, path)
				}

				if !test.expectedClaims {
					return
				}

				assert.Equal(t, test.expectedIdentities, data.Substrate["Identities"])

				identities, err := main.GetStringSlice("locals.identities")
				require.NoError(t, err)
				assert.Equal(t, test.expectedIdentities, identities)

				for _, path := range []string{"data.aws_instances.persistent.instance_tags", "data.aws_ebs_volumes.persistent.tags"} {
					tags, err := main.GetBytes(path)
					require.NoError(t, err, path)
					assert.JSONEq(t, `{"foundry.signoz.io/name":"foundry","foundry.signoz.io/storage":"persistent"}`, string(tags), path)
				}

				condition, err := main.GetBytes("data.aws_ebs_volumes.persistent.lifecycle.postcondition.0.condition")
				require.NoError(t, err)
				assert.Equal(t, "${length(self.ids) > 0}", string(condition))

				key, err := main.GetBytes("resource.aws_ec2_tag.claims.key")
				require.NoError(t, err)
				assert.Equal(t, "foundry.signoz.io/identities", string(key))

				preventDestroy, err := main.GetBytes("resource.aws_ec2_tag.claims.lifecycle.prevent_destroy")
				require.NoError(t, err)
				assert.Equal(t, "true", string(preventDestroy))

				return
			}

			require.Error(t, err)
		})
	}
}

var claimsTest = template.Must(template.New("claims.tftest.hcl").Funcs(template.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
}).Parse(`mock_provider "aws" {}

override_data {
  target = data.aws_subnets.substrate
  values = {
    ids = ["subnet-0a1b2c3d"]
  }
}

override_data {
  target = data.aws_instances.persistent
  values = {
    ids = {{ json .Instances }}
  }
}
{{ range $id, $volumes := .Attached }}
override_data {
  target = data.aws_instance.persistent["{{ $id }}"]
  values = {
    ebs_block_device = [{{ range $i, $volume := $volumes }}{{ if $i }}, {{ end }}{ volume_id = "{{ $volume }}" }{{ end }}]
  }
}
{{ end }}
override_data {
  target = data.aws_ebs_volumes.persistent
  values = {
    ids = {{ json .VolumeIDs }}
  }
}
{{ range .Volumes }}
override_data {
  target = data.aws_ebs_volume.persistent["{{ .ID }}"]
  values = {
    tags = {{ if .Claim }}{ "foundry.signoz.io/identities" = "{{ .Claim }}" }{{ else }}{}{{ end }}
  }
}
{{ end }}
run "claims" {
  command = plan
{{ if .Pass }}
  assert {
    condition     = jsonencode(local.volumes) == jsonencode({{ json .ExpectedVolumes }})
    error_message = "volumes are ${jsonencode(local.volumes)}"
  }

  assert {
    condition     = jsonencode(local.seats) == jsonencode({{ json .ExpectedSeats }})
    error_message = "seats are ${jsonencode(local.seats)}"
  }

  assert {
    condition     = jsonencode({ for volume, claim in aws_ec2_tag.claims : volume => claim.value }) == jsonencode({{ json .ExpectedClaims }})
    error_message = "claims are ${jsonencode({ for volume, claim in aws_ec2_tag.claims : volume => claim.value })}"
  }
{{ end -}}
}
`))

func TestClaimsAssignEveryIdentityAVolume(t *testing.T) {
	requireTerraform(t)

	const (
		keeper0   = "telemetrykeeper-clickhousekeeper-0"
		keeper1   = "telemetrykeeper-clickhousekeeper-1"
		keeper2   = "telemetrykeeper-clickhousekeeper-2"
		metastore = "metastore-postgres-0"
		store     = "telemetrystore-clickhouse-0-0"
	)

	bound := moldedCasting(t, boundCasting(&installation.Casting{}))

	keepers := boundCasting(&installation.Casting{})
	keepers.Spec.TelemetryKeeper.Spec.Cluster.Replicas = v1alpha1.IntPtr(3)
	keepers = moldedCasting(t, keepers)

	type volume struct {
		ID    string
		Claim string
	}

	// Five nodes over three volumes leave keeper 1 beside the store and keeper
	// 2 alone, which is what scaling the keepers down to one finds.
	scaledDown := []volume{{"vol-a", metastore + "," + keeper0}, {"vol-b", keeper1 + "," + store}, {"vol-c", keeper2}}

	tests := []struct {
		name            string
		casting         *installation.Casting
		volumes         []volume
		attached        map[string][]string
		expectedVolumes map[string]string
		expectedSeats   map[string]string
		expectedClaims  map[string]string
		expectedMessage string
		pass            bool
	}{
		{
			name:            "FirstAllocation_SortedUnclaimed",
			casting:         bound,
			volumes:         []volume{{ID: "vol-c"}, {ID: "vol-a"}, {ID: "vol-b"}},
			attached:        map[string][]string{"i-1": {"vol-a", "vol-b"}, "i-2": {"vol-c"}},
			expectedVolumes: map[string]string{keeper0: "vol-a", metastore: "vol-b", store: "vol-c"},
			expectedSeats:   map[string]string{keeper0: "i-1", metastore: "i-1", store: "i-2"},
			expectedClaims:  map[string]string{"vol-a": keeper0, "vol-b": metastore, "vol-c": store},
			pass:            true,
		},
		{
			name:            "MoreIdentitiesThanVolumes_Shared",
			casting:         bound,
			volumes:         []volume{{ID: "vol-b"}, {ID: "vol-a"}},
			attached:        map[string][]string{"i-1": {"vol-a"}, "i-2": {"vol-b"}},
			expectedVolumes: map[string]string{keeper0: "vol-a", metastore: "vol-b", store: "vol-a"},
			expectedSeats:   map[string]string{keeper0: "i-1", metastore: "i-2", store: "i-1"},
			expectedClaims:  map[string]string{"vol-a": keeper0 + "," + store, "vol-b": metastore},
			pass:            true,
		},
		{
			name:            "PartialClaimsRetried_Stable",
			casting:         bound,
			volumes:         []volume{{ID: "vol-a"}, {"vol-b", metastore}, {ID: "vol-c"}},
			attached:        map[string][]string{"i-1": {"vol-a", "vol-b"}},
			expectedVolumes: map[string]string{keeper0: "vol-a", metastore: "vol-b", store: "vol-c"},
			expectedSeats:   map[string]string{keeper0: "i-1", metastore: "i-1", store: "unattached"},
			expectedClaims:  map[string]string{"vol-a": keeper0, "vol-b": metastore, "vol-c": store},
			pass:            true,
		},
		{
			name:            "Shrink_RemovedClaimKept",
			casting:         bound,
			volumes:         scaledDown,
			attached:        map[string][]string{"i-1": {"vol-a", "vol-b", "vol-c"}},
			expectedVolumes: map[string]string{keeper0: "vol-a", metastore: "vol-a", store: "vol-b"},
			expectedSeats:   map[string]string{keeper0: "i-1", metastore: "i-1", store: "i-1"},
			expectedClaims:  map[string]string{"vol-a": metastore + "," + keeper0, "vol-b": keeper1 + "," + store},
			pass:            true,
		},
		{
			name:            "GrowAfterShrink_ReturnsToItsVolume",
			casting:         keepers,
			volumes:         scaledDown,
			attached:        map[string][]string{"i-1": {"vol-a", "vol-b", "vol-c"}},
			expectedVolumes: map[string]string{keeper0: "vol-a", keeper1: "vol-b", keeper2: "vol-c", metastore: "vol-a", store: "vol-b"},
			expectedSeats:   map[string]string{keeper0: "i-1", keeper1: "i-1", keeper2: "i-1", metastore: "i-1", store: "i-1"},
			expectedClaims:  map[string]string{"vol-a": metastore + "," + keeper0, "vol-b": keeper1 + "," + store, "vol-c": keeper2},
			pass:            true,
		},
		{
			name:            "NoVolume_Invalid",
			casting:         bound,
			attached:        map[string][]string{"i-1": {}},
			expectedMessage: "infrastructure foundry has no persistent volume to claim.",
		},
	}

	cache := t.TempDir()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			materials, err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *test.casting, "")
			require.NoError(t, err)

			pours := t.TempDir()

			w, err := writer.New(slog.New(slog.DiscardHandler), &writer.Options{Output: &os.File{}, TargetDirectory: pours})
			require.NoError(t, err)
			require.NoError(t, w.WriteMany(context.Background(), materials...))

			root := filepath.Join(pours, rootcasting.DeploymentDir)

			volumeIDs := []string{}
			for _, volume := range test.volumes {
				volumeIDs = append(volumeIDs, volume.ID)
			}

			hcl := bytes.NewBuffer(nil)
			require.NoError(t, claimsTest.Execute(hcl, map[string]any{
				"Instances":       slices.Sorted(maps.Keys(test.attached)),
				"Attached":        test.attached,
				"VolumeIDs":       volumeIDs,
				"Volumes":         test.volumes,
				"Pass":            test.pass,
				"ExpectedVolumes": test.expectedVolumes,
				"ExpectedSeats":   test.expectedSeats,
				"ExpectedClaims":  test.expectedClaims,
			}))
			require.NoError(t, os.WriteFile(filepath.Join(root, "claims.tftest.hcl"), hcl.Bytes(), 0o644))

			out, err := terraform(t, root, cache, "init", "-backend=false", "-input=false", "-no-color")
			require.NoError(t, err, out)

			out, err = terraform(t, root, cache, "test", "-no-color")

			if test.pass {
				require.NoError(t, err, out)

				return
			}

			require.Error(t, err, out)
			assert.Contains(t, out, test.expectedMessage)
		})
	}
}
