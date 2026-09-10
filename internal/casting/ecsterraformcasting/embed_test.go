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

// The cluster is placed onto, not provisioned, so every object it is made of is
// stated and nothing renders without them.
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

// The region is the one axis with no object to name, so it is all the tfvars hold.
func TestTfvarsTemplateCarriesTheRegion(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	require.NoError(t, tfarsTF.Execute(buf, templateDataFor(t, statedCasting(&installation.Casting{}))))

	assert.JSONEq(t, `{"aws_region": "us-east-1"}`, buf.String())
}

// Each object is named once, in a variable, and every component reads the local
// that resolves it. Nothing is looked up, so the root holds no data source --
// and an empty "data" object is a root terraform refuses outright.
func TestStatedObjectsAreResolvedThroughLocals(t *testing.T) {
	data := templateDataFor(t, statedCasting(&installation.Casting{}))

	variables := bytes.NewBuffer(nil)
	require.NoError(t, variablesTF.Execute(variables, data))

	for _, expected := range []string{
		`"default": "arn:aws:ecs:us-east-1:123456789012:cluster/test"`,
		`"default": ["subnet-abc123","subnet-def456"]`,
		`"default": ["sg-abc123"]`,
		`"default": "vpc-abc123"`,
		`"default": "signoz-installation-iam-task"`,
		`"default": "signoz-installation-iam-exec"`,
	} {
		assert.Contains(t, variables.String(), expected)
	}

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

// A role the operator brought is referenced as-is, and this stack creates none.
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

	without := func(key string) map[string]string {
		annotations := map[string]string{}
		maps.Copy(annotations, complete)
		delete(annotations, key)

		return annotations
	}

	malformed := map[string]string{}
	maps.Copy(malformed, complete)
	malformed[installation.ECSSubnetIDs.Key] = " , ,"

	tests := []struct {
		name        string
		annotations map[string]string
		pass        bool
	}{
		{name: "AllStated_Valid", annotations: complete, pass: true},
		{name: "RegionUnstated_Invalid", annotations: without(installation.ECSRegion.Key)},
		{name: "ClusterUnstated_Invalid", annotations: without(installation.ECSClusterARN.Key)},
		{name: "VPCUnstated_Invalid", annotations: without(installation.ECSVPCID.Key)},
		{name: "SubnetIDsUnstated_Invalid", annotations: without(installation.ECSSubnetIDs.Key)},
		{name: "SecurityGroupIDsUnstated_Invalid", annotations: without(installation.ECSSecurityGroupIDs.Key)},
		{name: "SubnetIDsMalformed_Invalid", annotations: malformed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := installation.Default(&installation.Casting{})
			casting.Metadata.Annotations = tt.annotations

			data, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "us-east-1", data.Region)

			for _, reference := range []Reference{data.Cluster, data.VPC, data.Subnets, data.SecurityGroup} {
				assert.True(t, reference.IsStated())
			}

			// The roles are this stack's own, so an unstated one is created.
			assert.False(t, data.TaskRole.IsStated())
		})
	}
}

// Binding a substrate reaches nothing in this casting, so it is refused rather
// than read by nothing.
func TestSubstrateBound_Invalid(t *testing.T) {
	casting := statedCasting(&installation.Casting{})
	casting.Spec.Infrastructure.Name = "signoz-infra"

	_, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
	assert.Error(t, err)
}

// sqlite is a file the signoz task holds, so no metastore service is forged and
// no output may name one.
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

// The agent attributes a log line by the labels docker copies into it, so a
// container that writes through another driver, or through json-file without
// the labels, reaches SigNoz carrying only a container id. Fargate refuses
// json-file, so the migrator is the one task that carries no block.
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

// A revision that never becomes healthy would otherwise sit there, because
// nothing else puts the previous one back.
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

// The application carries the Kind, so a CollectionAgent of the same name on
// the same account holds its own.
func TestAppConfigApplicationCarriesTheKind(t *testing.T) {
	data := templateDataFor(t, statedCasting(&installation.Casting{}))

	main := bytes.NewBuffer(nil)
	require.NoError(t, mainTF.Execute(main, data))

	material, err := domain.NewJSONMaterial(main.Bytes(), "main.tf.json")
	require.NoError(t, err)

	for path, expected := range map[string]string{
		"resource.aws_appconfig_application.main.name":         "signoz-installation-appconfig",
		"resource.aws_appconfig_deployment_strategy.main.name": "signoz-installation-appconfig-strategy",

		// The namespace is a DNS name components resolve each other by, not a
		// resource name, so it does not follow.
		"resource.aws_service_discovery_private_dns_namespace.main.name": "signoz.local",
	} {
		value, err := material.GetBytes(path)

		assert.NoError(t, err, "reading %s", path)
		assert.Equal(t, expected, string(value), "at %s", path)
	}

	// The sidecar prefetches by the application name, so a rename that misses
	// the triple leaves it fetching a profile that does not exist.
	ingester := bytes.NewBuffer(nil)
	require.NoError(t, ingesterTF.Execute(ingester, data))

	assert.Contains(t, ingester.String(), "signoz-installation-appconfig:default:ingester")
}
