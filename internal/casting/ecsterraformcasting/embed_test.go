package ecsterraformcasting

import (
	"bytes"
	"context"
	"log/slog"
	"maps"
	"strings"
	"testing"

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
		installation.ECSSubnetIDs.Key:        "subnet-abc123, subnet-def456",
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

func TestNotEmptyAndValid(t *testing.T) {
	templates := map[string]*domain.Template{
		"versionsTF":        versionsTF,
		"backendTF":         backendTF,
		"providersTF":       providersTF,
		"mainTF":            mainTF,
		"variablesTF":       variablesTF,
		"outputsTF":         outputsTF,
		"telemetryKeeperTF": telemetryKeeperTF,
		"telemetryStoreTF":  telemetryStoreTF,
		"migratorTF":        migratorTF,
		"metaStoreTF":       metaStoreTF,
		"signozTF":          signozTF,
		"ingesterTF":        ingesterTF,
		"mcpTF":             mcpTF,
	}

	for name, tmpl := range templates {
		assert.NotEmpty(t, tmpl, "%s should not be empty", name)
		buf := bytes.NewBuffer(nil)
		err := tmpl.Execute(buf, nil)
		assert.NoError(t, err, "error executing %s", name)
		assert.NotEmpty(t, buf.String(), "%s output should not be empty", name)
	}
}

// A default in the variables would race the tfvars.
func TestTfvarsTemplateCarriesEveryValue(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	require.NoError(t, tfarsTF.Execute(buf, templateDataFor(t, statedCasting(&installation.Casting{}))))

	assert.JSONEq(t, `{
		"aws_region": "us-east-1",
		"cluster_arn": "arn:aws:ecs:us-east-1:123456789012:cluster/test",
		"subnet_ids": ["subnet-abc123", "subnet-def456"],
		"security_group_ids": ["sg-abc123"],
		"vpc_id": "vpc-abc123",
		"task_role_name": "signoz-installation-iam-task",
		"execution_role_name": "signoz-installation-iam-exec"
	}`, buf.String())
}

// No name is computed for a role this stack does not create.
func TestTfvarsTemplateCarriesStatedRoles(t *testing.T) {
	casting := statedCasting(&installation.Casting{})
	casting.Metadata.Annotations[installation.ECSTaskRoleARN.Key] = "arn:aws:iam::123456789012:role/task"
	casting.Metadata.Annotations[installation.ECSTaskExecutionRoleARN.Key] = "arn:aws:iam::123456789012:role/exec"

	buf := bytes.NewBuffer(nil)
	require.NoError(t, tfarsTF.Execute(buf, templateDataFor(t, casting)))

	assert.JSONEq(t, `{
		"aws_region": "us-east-1",
		"cluster_arn": "arn:aws:ecs:us-east-1:123456789012:cluster/test",
		"subnet_ids": ["subnet-abc123", "subnet-def456"],
		"security_group_ids": ["sg-abc123"],
		"vpc_id": "vpc-abc123",
		"task_role_arn": "arn:aws:iam::123456789012:role/task",
		"execution_role_arn": "arn:aws:iam::123456789012:role/exec"
	}`, buf.String())
}

// An empty "data" object is a root terraform refuses outright.
func TestStatedObjectsAreResolvedThroughLocals(t *testing.T) {
	data := templateDataFor(t, statedCasting(&installation.Casting{}))

	variables := bytes.NewBuffer(nil)
	require.NoError(t, variablesTF.Execute(variables, data))

	for _, expected := range []string{
		`"cluster_arn"`,
		`"subnet_ids"`,
		`"security_group_ids"`,
		`"vpc_id"`,
		`"task_role_name"`,
		`"execution_role_name"`,
	} {
		assert.Contains(t, variables.String(), expected)
	}

	assert.NotContains(t, variables.String(), `"default"`)

	main := bytes.NewBuffer(nil)
	require.NoError(t, mainTF.Execute(main, data))

	out := main.String()
	for _, expected := range []string{
		`"cluster_arn": "${var.cluster_arn}"`,
		`"subnet_ids": "${var.subnet_ids}"`,
		`"security_group_ids": "${var.security_group_ids}"`,
		`"vpc_id": "${var.vpc_id}"`,
		`"name": "${var.task_role_name}"`,
	} {
		assert.Contains(t, out, expected)
	}

	material, err := domain.NewJSONMaterial(main.Bytes(), "main.tf.json")
	require.NoError(t, err)

	_, err = material.GetBytes("data")
	assert.Error(t, err, "a root with no lookups emits no data block")
}

func TestStatedRoleIsNotCreated(t *testing.T) {
	casting := statedCasting(&installation.Casting{})
	casting.Metadata.Annotations[installation.ECSTaskRoleARN.Key] = "arn:aws:iam::123456789012:role/task"
	casting.Metadata.Annotations[installation.ECSTaskExecutionRoleARN.Key] = "arn:aws:iam::123456789012:role/exec"

	data := templateDataFor(t, casting)

	main := bytes.NewBuffer(nil)
	require.NoError(t, mainTF.Execute(main, data))

	out := main.String()
	assert.NotContains(t, out, "aws_iam_role")
	assert.NotContains(t, out, "appconfig:StartConfigurationSession")
	assert.Contains(t, out, `"task_role_arn": "${var.task_role_arn}"`)
}

func TestReferenceIsStated(t *testing.T) {
	tests := []struct {
		name      string
		reference Reference
		pass      bool
	}{
		{name: "Stated_Valid", reference: Reference{Stated: "arn:aws:ecs:us-east-1:1:cluster/x"}, pass: true},
		{name: "StatedIDs_Valid", reference: Reference{StatedIDs: []string{"subnet-a"}}, pass: true},
		{name: "Empty_Invalid", reference: Reference{}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.pass, tt.reference.IsStated())
		})
	}
}

func TestTemplateDataResolution(t *testing.T) {
	complete := statedCasting(&installation.Casting{}).Metadata.Annotations

	malformed := map[string]string{}
	maps.Copy(malformed, complete)
	malformed[installation.ECSSubnetIDs.Key] = " , ,"

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

// The platform judges its own inputs, so an unstated casting still forges.
func TestUnstatedCastingForges(t *testing.T) {
	casting := installation.Default(&installation.Casting{})

	materials, err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *casting, "")
	require.NoError(t, err)
	assert.NotEmpty(t, materials)

	buf := bytes.NewBuffer(nil)
	require.NoError(t, tfarsTF.Execute(buf, templateDataFor(t, casting)))

	assert.JSONEq(t, `{
		"aws_region": "",
		"cluster_arn": "",
		"subnet_ids": [],
		"security_group_ids": [],
		"vpc_id": "",
		"task_role_name": "signoz-installation-iam-task",
		"execution_role_name": "signoz-installation-iam-exec"
	}`, buf.String())
}

// Terraform refuses an empty or malformed value at plan, so every variable the
// casting does not judge carries its own condition.
func TestVariablesCarryValidations(t *testing.T) {
	casting := statedCasting(&installation.Casting{})
	casting.Metadata.Annotations[installation.ECSTaskRoleARN.Key] = "arn:aws:iam::123456789012:role/task"
	casting.Metadata.Annotations[installation.ECSTaskExecutionRoleARN.Key] = "arn:aws:iam::123456789012:role/exec"

	buf := bytes.NewBuffer(nil)
	require.NoError(t, variablesTF.Execute(buf, templateDataFor(t, casting)))

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

// Sqlite is a file the signoz task holds, so it is no service of its own.
func TestSqliteForgesNoMetaStoreService(t *testing.T) {
	casting := statedCasting(&installation.Casting{})
	casting.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite

	materials, err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *casting, "")
	require.NoError(t, err)

	for _, material := range materials {
		assert.NotContains(t, material.Path(), "metastore.tf.json")
	}

	outputs := bytes.NewBuffer(nil)
	require.NoError(t, outputsTF.Execute(outputs, templateDataFor(t, casting)))

	assert.NotContains(t, outputs.String(), "metastore_service_name")
}

func TestPostgresForgesAMetaStoreService(t *testing.T) {
	casting := statedCasting(&installation.Casting{})
	casting.Spec.MetaStore.Kind = installation.MetaStoreKindPostgres

	materials, err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *casting, "")
	require.NoError(t, err)

	forged := false
	for _, material := range materials {
		if strings.Contains(material.Path(), "metastore.tf.json") {
			forged = true
		}
	}

	assert.True(t, forged)

	outputs := bytes.NewBuffer(nil)
	require.NoError(t, outputsTF.Execute(outputs, templateDataFor(t, casting)))

	assert.Contains(t, outputs.String(), "metastore_service_name")
}

// A container that logs without the labels reaches SigNoz carrying only a
// container id. Fargate refuses json-file, so the migrator carries no block.
func TestEveryEC2ContainerLogsWithLabels(t *testing.T) {
	data := templateDataFor(t, statedCasting(&installation.Casting{}))

	for name, tmpl := range map[string]*domain.Template{
		"signozTF":          signozTF,
		"ingesterTF":        ingesterTF,
		"metaStoreTF":       metaStoreTF,
		"mcpTF":             mcpTF,
		"telemetryStoreTF":  telemetryStoreTF,
		"telemetryKeeperTF": telemetryKeeperTF,
	} {
		buf := bytes.NewBuffer(nil)
		require.NoError(t, tmpl.Execute(buf, data), name)

		rendered := buf.String()

		assert.Equal(t, strings.Count(rendered, `"image":`), strings.Count(rendered, `"logConfiguration":`),
			"%s: every container logs, or none of them is attributed", name)
		assert.Contains(t, rendered, "com.amazonaws.ecs.task-definition-family,com.amazonaws.ecs.container-name,com.amazonaws.ecs.task-arn,com.amazonaws.ecs.cluster", name)

		// Without rotation docker fills the instance disk.
		assert.Contains(t, rendered, `"max-size": "10m"`, name)
	}

	migrator := bytes.NewBuffer(nil)
	require.NoError(t, migratorTF.Execute(migrator, data))

	assert.NotContains(t, migrator.String(), "logConfiguration")
}

// Nothing else puts a previous revision back.
func TestEveryServiceRollsBackABadRevision(t *testing.T) {
	data := templateDataFor(t, statedCasting(&installation.Casting{}))

	for name, tmpl := range map[string]*domain.Template{
		"signozTF":          signozTF,
		"ingesterTF":        ingesterTF,
		"metaStoreTF":       metaStoreTF,
		"mcpTF":             mcpTF,
		"telemetryStoreTF":  telemetryStoreTF,
		"telemetryKeeperTF": telemetryKeeperTF,
	} {
		buf := bytes.NewBuffer(nil)
		require.NoError(t, tmpl.Execute(buf, data), name)

		rendered := buf.String()

		assert.Equal(t, strings.Count(rendered, `"launch_type": "EC2"`), strings.Count(rendered, `"deployment_circuit_breaker"`), name)
		assert.NotContains(t, rendered, "capacity_provider")
	}
}

// A CollectionAgent of the same name on the same account holds its own.
func TestAppConfigApplicationCarriesTheKind(t *testing.T) {
	data := templateDataFor(t, statedCasting(&installation.Casting{}))

	main := bytes.NewBuffer(nil)
	require.NoError(t, mainTF.Execute(main, data))

	material, err := domain.NewJSONMaterial(main.Bytes(), "main.tf.json")
	require.NoError(t, err)

	for path, expected := range map[string]string{
		"resource.aws_appconfig_application.main.name":         "signoz-installation-appconfig",
		"resource.aws_appconfig_deployment_strategy.main.name": "signoz-installation-appconfig-strategy",

		"resource.aws_service_discovery_private_dns_namespace.main.name": "signoz-installation.local",
	} {
		value, err := material.GetBytes(path)

		assert.NoError(t, err, "reading %s", path)
		assert.Equal(t, expected, string(value), "at %s", path)
	}

	// The sidecar prefetches by the application name, so a rename must reach it.
	ingester := bytes.NewBuffer(nil)
	require.NoError(t, ingesterTF.Execute(ingester, data))

	assert.Contains(t, ingester.String(), "signoz-installation-appconfig:default:ingester")
}

// A task bound to a disk cannot be replaced before the one holding it stops.
func TestServiceRollsBeforeStopping(t *testing.T) {
	tests := []struct {
		name                   string
		template               *domain.Template
		metaStoreKind          installation.MetaStoreKind
		service                string
		expectedMinimumPercent string
		expectedMaximumPercent string
	}{
		{name: "Ingester_Rolling", template: ingesterTF, metaStoreKind: installation.MetaStoreKindPostgres, service: "ingester", expectedMinimumPercent: "100", expectedMaximumPercent: "200"},
		{name: "MCP_Rolling", template: mcpTF, metaStoreKind: installation.MetaStoreKindPostgres, service: "mcp", expectedMinimumPercent: "100", expectedMaximumPercent: "200"},
		{name: "SignozOnPostgres_Rolling", template: signozTF, metaStoreKind: installation.MetaStoreKindPostgres, service: "signoz_0", expectedMinimumPercent: "100", expectedMaximumPercent: "200"},
		{name: "SignozOnSqlite_StopFirst", template: signozTF, metaStoreKind: installation.MetaStoreKindSQLite, service: "signoz_0", expectedMinimumPercent: "0", expectedMaximumPercent: "100"},
		{name: "TelemetryStore_StopFirst", template: telemetryStoreTF, metaStoreKind: installation.MetaStoreKindPostgres, service: "telemetrystore_clickhouse_0_0", expectedMinimumPercent: "0", expectedMaximumPercent: "100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := statedCasting(&installation.Casting{})
			casting.Spec.MetaStore.Kind = tt.metaStoreKind

			buf := bytes.NewBuffer(nil)
			require.NoError(t, tt.template.Execute(buf, templateDataFor(t, casting)))

			material, err := domain.NewJSONMaterial(buf.Bytes(), "service.tf.json")
			require.NoError(t, err)

			minimum, err := material.GetBytes("resource.aws_ecs_service." + tt.service + ".deployment_minimum_healthy_percent")
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMinimumPercent, string(minimum))

			maximum, err := material.GetBytes("resource.aws_ecs_service." + tt.service + ".deployment_maximum_percent")
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMaximumPercent, string(maximum))
		})
	}
}

// Replication reaches a node on the interserver port, and a node holding data
// takes longer to answer than the default start period allows.
func TestTelemetryStoreContainerIsReachableAndGivenTimeToStart(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	require.NoError(t, telemetryStoreTF.Execute(buf, templateDataFor(t, statedCasting(&installation.Casting{}))))

	material, err := domain.NewJSONMaterial(buf.Bytes(), "telemetrystore.tf.json")
	require.NoError(t, err)

	container := `locals.containers_telemetrystore_clickhouse_0_0.#(name=="signoz-telemetrystore-clickhouse-0-0")`

	tests := []struct {
		name          string
		path          string
		expectedValue string
	}{
		{name: "NativePort_Mapped", path: container + `.portMappings.#(containerPort==9000).name`, expectedValue: "native"},
		{name: "HTTPPort_Mapped", path: container + `.portMappings.#(containerPort==8123).name`, expectedValue: "http"},
		{name: "PrometheusPort_Mapped", path: container + `.portMappings.#(containerPort==9363).name`, expectedValue: "prometheus"},
		{name: "InterserverPort_Mapped", path: container + `.portMappings.#(containerPort==9009).name`, expectedValue: "interserver"},
		{name: "StartPeriod_Stated", path: container + `.healthCheck.startPeriod`, expectedValue: "300"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := material.GetBytes(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedValue, string(value))
		})
	}
}

// Nothing chowns the task volume, so the agent writes as root and the server
// stays root to read what it wrote.
func TestTelemetryStoreRunsAsRoot(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	require.NoError(t, telemetryStoreTF.Execute(buf, templateDataFor(t, statedCasting(&installation.Casting{}))))

	material, err := domain.NewJSONMaterial(buf.Bytes(), "telemetrystore.tf.json")
	require.NoError(t, err)

	agent := `locals.containers_telemetrystore_clickhouse_0_0.#(name=="signoz-telemetrystore-appconfig-agent")`
	server := `locals.containers_telemetrystore_clickhouse_0_0.#(name=="signoz-telemetrystore-clickhouse-0-0")`
	scripts := `locals.containers_telemetrystore_clickhouse_0_0.#(name=="signoz-telemetrystore-user-scripts")`

	t.Run("AgentUser_Unstated", func(t *testing.T) {
		_, err := material.GetBytes(agent + ".user")
		assert.Error(t, err)
	})

	t.Run("ServerRunAsRoot_Stated", func(t *testing.T) {
		value, err := material.GetBytes(server + `.environment.#(name=="CLICKHOUSE_RUN_AS_ROOT").value`)
		require.NoError(t, err)
		assert.Equal(t, "1", string(value))
	})

	t.Run("UserScriptsCommand_Chownless", func(t *testing.T) {
		value, err := material.GetBytes(scripts + ".command.0")
		require.NoError(t, err)
		assert.NotContains(t, string(value), "chown")
	})
}

// Nothing chowns the task volume, so the agent writes as root and the
// collector reads what it wrote.
func TestIngesterContainersRunAsRoot(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	require.NoError(t, ingesterTF.Execute(buf, templateDataFor(t, statedCasting(&installation.Casting{}))))

	material, err := domain.NewJSONMaterial(buf.Bytes(), "ingester.tf.json")
	require.NoError(t, err)

	tests := []struct {
		name          string
		path          string
		expectedValue string
	}{
		{name: "ContainerCount_Stated", path: "locals.containers_ingester.#", expectedValue: "2"},
		{name: "AgentName_Stated", path: "locals.containers_ingester.0.name", expectedValue: "signoz-ingester-appconfig-agent"},
		{name: "CollectorName_Stated", path: "locals.containers_ingester.1.name", expectedValue: "signoz-ingester"},
		{name: "AgentUser_Root", path: `locals.containers_ingester.#(name=="signoz-ingester-appconfig-agent").user`, expectedValue: "0"},
		{name: "CollectorUser_Root", path: `locals.containers_ingester.#(name=="signoz-ingester").user`, expectedValue: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := material.GetBytes(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedValue, string(value))
		})
	}
}

// The keeper writes a host path owned by root, and a node holding a raft log
// takes longer to answer than the default start period allows.
func TestTelemetryKeeperContainerRunsAsRootAndIsGivenTimeToStart(t *testing.T) {
	zookeeper := `locals.containers_telemetrykeeper_zookeeper_0.#(name=="signoz-telemetrykeeper-zookeeper-0")`
	clickhouseKeeper := `locals.containers_telemetrykeeper_clickhousekeeper_0.#(name=="signoz-telemetrykeeper-clickhousekeeper-0")`

	tests := []struct {
		name          string
		kind          installation.TelemetryKeeperKind
		path          string
		expectedValue string
	}{
		{name: "ZookeeperUser_Root", kind: installation.TelemetryKeeperKindZookeeper, path: zookeeper + ".user", expectedValue: "0"},
		{name: "ZookeeperPrometheusPort_Mapped", kind: installation.TelemetryKeeperKindZookeeper, path: zookeeper + `.portMappings.#(containerPort==9141).name`, expectedValue: "prometheus"},
		{name: "ZookeeperStartPeriod_Stated", kind: installation.TelemetryKeeperKindZookeeper, path: zookeeper + ".healthCheck.startPeriod", expectedValue: "300"},
		{name: "ClickhouseKeeperStartPeriod_Stated", kind: installation.TelemetryKeeperKindClickhouseKeeper, path: clickhouseKeeper + ".healthCheck.startPeriod", expectedValue: "300"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := statedCasting(&installation.Casting{})
			casting.Spec.TelemetryKeeper.Kind = tt.kind

			buf := bytes.NewBuffer(nil)
			require.NoError(t, telemetryKeeperTF.Execute(buf, templateDataFor(t, casting)))

			material, err := domain.NewJSONMaterial(buf.Bytes(), "telemetrykeeper.tf.json")
			require.NoError(t, err)

			value, err := material.GetBytes(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedValue, string(value))
		})
	}
}

// SigNoz carries a node ordinal whatever the metadata store is, so only the
// sqlite disk and its rollout tell the two apart.
func TestSignozIsAlwaysANode(t *testing.T) {
	tests := []struct {
		name                   string
		metaStoreKind          installation.MetaStoreKind
		expectedHostPath       string
		expectedMinimumPercent string
		expectedMaximumPercent string
	}{
		{
			name:                   "Postgres_Diskless",
			metaStoreKind:          installation.MetaStoreKindPostgres,
			expectedMinimumPercent: "100",
			expectedMaximumPercent: "200",
		},
		{
			name:                   "Sqlite_HoldsADisk",
			metaStoreKind:          installation.MetaStoreKindSQLite,
			expectedHostPath:       "/var/lib/foundry/signoz/metastore/sqlite/0",
			expectedMinimumPercent: "0",
			expectedMaximumPercent: "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := statedCasting(&installation.Casting{})
			casting.Spec.MetaStore.Kind = tt.metaStoreKind

			buf := bytes.NewBuffer(nil)
			require.NoError(t, signozTF.Execute(buf, templateDataFor(t, casting)))

			material, err := domain.NewJSONMaterial(buf.Bytes(), "signoz.tf.json")
			require.NoError(t, err)

			expected := map[string]string{
				"locals.containers_signoz_0.0.name":                                    "signoz-signoz-0",
				"locals.containers_signoz_0.0.healthCheck.startPeriod":                 "300",
				"resource.aws_ecs_task_definition.signoz_0.family":                     "signoz-signoz-0",
				"resource.aws_service_discovery_service.signoz_0.name":                 "signoz-0",
				"resource.aws_ecs_service.signoz_0.name":                               "signoz-signoz-0",
				"resource.aws_ecs_service.signoz_0.task_definition":                    "${aws_ecs_task_definition.signoz_0.arn}",
				"resource.aws_ecs_service.signoz_0.desired_count":                      "1",
				"resource.aws_ecs_service.signoz_0.deployment_minimum_healthy_percent": tt.expectedMinimumPercent,
				"resource.aws_ecs_service.signoz_0.deployment_maximum_percent":         tt.expectedMaximumPercent,
			}

			if tt.expectedHostPath != "" {
				expected["resource.aws_ecs_task_definition.signoz_0.volume.0.host_path"] = tt.expectedHostPath
			}

			for path, want := range expected {
				value, err := material.GetBytes(path)

				require.NoError(t, err, "reading %s", path)
				assert.Equal(t, want, string(value), "at %s", path)
			}

			if tt.expectedHostPath == "" {
				_, err := material.GetBytes("resource.aws_ecs_task_definition.signoz_0.volume")
				assert.Error(t, err)
			}

			outputs := bytes.NewBuffer(nil)
			require.NoError(t, outputsTF.Execute(outputs, templateDataFor(t, casting)))

			assert.Contains(t, outputs.String(), "${aws_ecs_service.signoz_0.name}")
			assert.Contains(t, outputs.String(), "${aws_ecs_service.signoz_0.id}")
		})
	}
}

// A postgres holding a data directory takes longer to answer than the default
// start period allows.
func TestMetaStoreIsANodeGivenTimeToStart(t *testing.T) {
	casting := statedCasting(&installation.Casting{})
	casting.Spec.MetaStore.Kind = installation.MetaStoreKindPostgres

	buf := bytes.NewBuffer(nil)
	require.NoError(t, metaStoreTF.Execute(buf, templateDataFor(t, casting)))

	material, err := domain.NewJSONMaterial(buf.Bytes(), "metastore.tf.json")
	require.NoError(t, err)

	tests := []struct {
		name          string
		path          string
		expectedValue string
	}{
		{name: "StartPeriod_Stated", path: "locals.containers_metastore_postgres_0.0.healthCheck.startPeriod", expectedValue: "300"},
		{name: "ContainerName_Stated", path: "locals.containers_metastore_postgres_0.0.name", expectedValue: "signoz-metastore-postgres-0"},
		{name: "Family_Stated", path: "resource.aws_ecs_task_definition.metastore_postgres_0.family", expectedValue: "signoz-metastore-postgres-0"},
		{name: "DNS_Stated", path: "resource.aws_service_discovery_service.metastore_postgres_0.name", expectedValue: "metastore-postgres-0"},
		{name: "ServiceName_Stated", path: "resource.aws_ecs_service.metastore_postgres_0.name", expectedValue: "signoz-metastore-postgres-0"},
		{name: "HostPath_Stated", path: "resource.aws_ecs_task_definition.metastore_postgres_0.volume.0.host_path", expectedValue: "/var/lib/foundry/signoz/metastore/postgres/0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := material.GetBytes(tt.path)
			require.NoError(t, err, "reading %s", tt.path)
			assert.Equal(t, tt.expectedValue, string(value))
		})
	}
}
