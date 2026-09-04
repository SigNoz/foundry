package ecsterraformcasting

import (
	"bytes"
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

// The binding is what the casting derives its lookup tags from, so nothing
// renders without it.
func boundCasting(declared *installation.Casting) *installation.Casting {
	c := installation.Default(declared)
	c.Spec.Infrastructure.Name = "signoz"
	c.Metadata.Annotations = map[string]string{installation.ECSRegion.Key: "us-east-1"}

	return c
}

func clusteredCasting(keeperKind installation.TelemetryKeeperKind, shards, replicas int) *installation.Casting {
	declared := &installation.Casting{}
	declared.Spec.TelemetryKeeper.Kind = keeperKind

	c := boundCasting(declared)
	c.Spec.TelemetryStore.Spec.Cluster.Shards = v1alpha1.IntPtr(shards)
	c.Spec.TelemetryStore.Spec.Cluster.Replicas = v1alpha1.IntPtr(replicas)
	c.Spec.TelemetryKeeper.Spec.Cluster.Replicas = v1alpha1.IntPtr(3)

	return c
}

func TestNotEmptyAndValid(t *testing.T) {
	templates := map[string]*domain.Template{
		"versionsTF":        versionsTF,
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

// The region is neither declared nor discoverable, so it is all the tfvars hold.
func TestTfvarsTemplateCarriesTheRegion(t *testing.T) {
	casting := boundCasting(&installation.Casting{})
	casting.Metadata.Annotations = map[string]string{installation.ECSRegion.Key: "us-east-1"}

	buf := bytes.NewBuffer(nil)
	require.NoError(t, tfarsTF.Execute(buf, templateDataFor(t, casting)))

	assert.JSONEq(t, `{"aws_region": "us-east-1"}`, buf.String())
}

// Nothing the substrate provisioned is named twice: the variables carry what
// the other casting derived, and main.tf looks the objects up by them.
func TestSubstrateIsLookedUpThroughVariables(t *testing.T) {
	data := templateDataFor(t, boundCasting(&installation.Casting{}))

	variables := bytes.NewBuffer(nil)
	require.NoError(t, variablesTF.Execute(variables, data))

	for _, expected := range []string{
		`"default": "signoz-cls"`,
		`"default": "signoz-sg-task"`,
		`"default": "signoz-installation-iam-task"`,
		`"default": "signoz-installation-iam-exec"`,
		`"foundry.signoz.io/subnet-type":"private"`,
		`"foundry.signoz.io/storage":"persistent"`,
	} {
		assert.Contains(t, variables.String(), expected)
	}

	main := bytes.NewBuffer(nil)
	require.NoError(t, mainTF.Execute(main, data))

	for _, expected := range []string{
		`"cluster_name": "${var.cluster_name}"`,
		`"tags": "${var.subnet_tags}"`,
		`"instance_tags": "${var.node_tags}"`,
		`"cluster_arn": "${data.aws_ecs_cluster.main.arn}"`,
		`"subnet_ids": "${data.aws_subnets.private.ids}"`,
		`volume.tags[var.claim_tag]`,
	} {
		assert.Contains(t, main.String(), expected)
	}
}

// An operator who runs a cluster foundry did not provision states the
// identifiers, and no lookup is emitted for them.
func TestStatedIdentifiersReplaceTheirLookup(t *testing.T) {
	casting := boundCasting(&installation.Casting{})
	casting.Metadata.Annotations[installation.ECSClusterARN.Key] = "arn:aws:ecs:us-east-1:123456789012:cluster/test"
	casting.Metadata.Annotations[installation.ECSSubnetIDs.Key] = "subnet-abc123, subnet-def456"

	data := templateDataFor(t, casting)

	variables := bytes.NewBuffer(nil)
	require.NoError(t, variablesTF.Execute(variables, data))

	assert.Contains(t, variables.String(), `"default": "arn:aws:ecs:us-east-1:123456789012:cluster/test"`)
	assert.Contains(t, variables.String(), `"default": ["subnet-abc123","subnet-def456"]`)

	main := bytes.NewBuffer(nil)
	require.NoError(t, mainTF.Execute(main, data))

	out := main.String()
	assert.Contains(t, out, `"cluster_arn": "${var.cluster_arn}"`)
	assert.Contains(t, out, `"subnet_ids": "${var.subnet_ids}"`)
	assert.NotContains(t, out, "aws_ecs_cluster")
	assert.NotContains(t, out, "aws_subnets")

	// The ones they did not state are still discovered.
	assert.Contains(t, out, `"name": "${var.task_role_name}"`)
	assert.Contains(t, variables.String(), `"default": "signoz-installation-iam-task"`)
}

func templateDataFor(t *testing.T, casting *installation.Casting) templateData {
	t.Helper()

	data, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
	require.NoError(t, err)

	return data
}

func TestModulePlacement(t *testing.T) {
	t.Parallel()

	sqlite := boundCasting(&installation.Casting{})
	sqlite.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite

	// The attribute clauses match what the substrate stamps on its nodes, so a
	// component reaches only the nodes of the substrate it is bound to.
	seat := func(identity string) string {
		return `ec2InstanceId == '${local.seats[\"` + identity + `\"]}' and attribute:foundry.signoz.io/name == signoz and attribute:foundry.signoz.io/storage == persistent`
	}
	ephemeral := "attribute:foundry.signoz.io/name == signoz and attribute:foundry.signoz.io/storage == ephemeral"

	// A stateful identity pins to the instance its claim resolved; the storage
	// attribute stays as a bootstrap integrity check.
	testCases := []struct {
		name                string
		template            *domain.Template
		casting             *installation.Casting
		expectedExpressions []string
	}{
		{
			name:                "TelemetryKeeper_PinnedToClaimedSeats",
			template:            telemetryKeeperTF,
			casting:             clusteredCasting(installation.TelemetryKeeperKindClickhouseKeeper, 2, 1),
			expectedExpressions: []string{seat("telemetrykeeper-0"), seat("telemetrykeeper-2")},
		},
		{
			name:                "TelemetryStore_PinnedToClaimedSeats",
			template:            telemetryStoreTF,
			casting:             clusteredCasting(installation.TelemetryKeeperKindClickhouseKeeper, 2, 1),
			expectedExpressions: []string{seat("telemetrystore-0-0"), seat("telemetrystore-1-1")},
		},
		{
			name:                "Metastore_PinnedToClaimedSeat",
			template:            metaStoreTF,
			casting:             boundCasting(&installation.Casting{}),
			expectedExpressions: []string{seat("metastore-0")},
		},
		{
			name:                "Ingester_Ephemeral",
			template:            ingesterTF,
			casting:             boundCasting(&installation.Casting{}),
			expectedExpressions: []string{ephemeral},
		},
		{
			name:                "SignozPostgres_Ephemeral",
			template:            signozTF,
			casting:             boundCasting(&installation.Casting{}),
			expectedExpressions: []string{ephemeral},
		},
		{
			name:                "SignozSqlite_PinnedToClaimedSeat",
			template:            signozTF,
			casting:             sqlite,
			expectedExpressions: []string{seat("signoz-0")},
		},
		{
			name:                "MCP_Ephemeral",
			template:            mcpTF,
			casting:             boundCasting(&installation.Casting{}),
			expectedExpressions: []string{ephemeral},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buf := bytes.NewBuffer(nil)
			require.NoError(t, tc.template.Execute(buf, templateDataFor(t, tc.casting)))
			out := buf.String()

			assert.Contains(t, out, `"launch_type": "EC2"`)
			for _, expression := range tc.expectedExpressions {
				assert.Contains(t, out, expression)
			}
			assert.NotContains(t, out, "capacity_provider")
		})
	}
}

func TestReferenceIsStated(t *testing.T) {
	tests := []struct {
		name      string
		reference Reference
		pass      bool
	}{
		{name: "Stated_Valid", reference: Reference{Stated: "arn:aws:ecs:us-east-1:1:cluster/x"}, pass: true},
		{name: "StatedIDs_Valid", reference: Reference{StatedIDs: []string{"subnet-a"}}, pass: true},
		{name: "Derived_Invalid", reference: Reference{Name: "signoz-cls", Tags: map[string]string{"a": "b"}}, pass: false},
		{name: "Empty_Invalid", reference: Reference{}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.pass, tt.reference.IsStated())
		})
	}
}

func TestTemplateDataResolution(t *testing.T) {
	stated := map[string]string{
		installation.ECSClusterARN.Key:       "arn:aws:ecs:us-east-1:1:cluster/mine",
		installation.ECSVPCID.Key:            "vpc-abc",
		installation.ECSSubnetIDs.Key:        "subnet-a, subnet-b",
		installation.ECSSecurityGroupIDs.Key: "sg-a",
	}

	tests := []struct {
		name        string
		substrate   string
		annotations map[string]string
		pass        bool
		expectedAll bool
	}{
		{name: "AllDerived_Valid", substrate: "signoz", pass: true},
		{name: "AllStated_Valid", substrate: "signoz", annotations: stated, pass: true, expectedAll: true},
		{name: "BYOAllStated_Valid", annotations: stated, pass: true, expectedAll: true},
		{name: "BYOClusterUnstated_Invalid", annotations: map[string]string{installation.ECSVPCID.Key: "vpc-abc"}},
		{name: "SubnetIDsMalformed_Invalid", substrate: "signoz", annotations: map[string]string{installation.ECSSubnetIDs.Key: " , ,"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := installation.Default(&installation.Casting{})
			casting.Spec.Infrastructure.Name = tt.substrate
			casting.Metadata.Annotations = map[string]string{installation.ECSRegion.Key: "us-east-1"}
			maps.Copy(casting.Metadata.Annotations, tt.annotations)

			data, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "us-east-1", data.Region)

			for _, reference := range []Reference{data.Cluster, data.VPC, data.Subnets, data.SecurityGroup} {
				assert.Equal(t, tt.expectedAll, reference.IsStated())
			}

			// The roles never come from the substrate, so the template names
			// them either way.
			assert.False(t, data.TaskRole.IsStated())
		})
	}
}

func TestRegionUnstated_Invalid(t *testing.T) {
	casting := installation.Default(&installation.Casting{})
	casting.Spec.Infrastructure.Name = "signoz"

	_, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
	assert.Error(t, err)
}

// The agent attributes a log line by the labels docker copies into it, so a
// container that writes through another driver, or through json-file without
// the labels, reaches SigNoz carrying only a container id. Fargate refuses
// json-file, so the migrator is the one task that carries no block.
func TestEveryEC2ContainerLogsWithLabels(t *testing.T) {
	data := templateDataFor(t, clusteredCasting(installation.TelemetryKeeperKindClickhouseKeeper, 1, 1))

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
	data := templateDataFor(t, clusteredCasting(installation.TelemetryKeeperKindClickhouseKeeper, 2, 2))

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
	}
}

// The application carries the Kind, so a CollectionAgent of the same name on
// the same account holds its own.
func TestAppConfigApplicationCarriesTheKind(t *testing.T) {
	data := templateDataFor(t, boundCasting(&installation.Casting{}))

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
