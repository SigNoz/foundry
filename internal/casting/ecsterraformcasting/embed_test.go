package ecsterraformcasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
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

func templateDataFor(t *testing.T, casting *installation.Casting) templateData {
	t.Helper()

	data, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
	require.NoError(t, err)

	return data
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

	tests := []struct {
		name            string
		annotations     map[string]string
		expectedRegion  string
		expectedStated  bool
		expectedMessage string
		pass            bool
	}{
		{name: "AllStated_Valid", annotations: complete, expectedRegion: "us-east-1", expectedStated: true, pass: true},
		{name: "NothingStated_Valid", annotations: nil, pass: true},
		{name: "SubnetIDsMalformed_Invalid", annotations: malformed, expectedMessage: "no ids found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := installation.Default(&installation.Casting{})
			casting.Metadata.Annotations = tt.annotations

			data, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
			if !tt.pass {
				assert.ErrorContains(t, err, tt.expectedMessage)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedRegion, data.Region)

			for _, reference := range []Reference{data.Cluster, data.VPC, data.Subnets, data.SecurityGroup} {
				assert.Equal(t, tt.expectedStated, reference.IsStated())
			}

			assert.False(t, data.TaskRole.IsStated())
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
