package ecsterraformcasting

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// boundCasting returns a fully-defaulted Installation bound to a substrate. The
// binding is what lets the casting derive the tags it looks resources up by, so
// nothing renders without it.
func boundCasting(declared *installation.Casting) *installation.Casting {
	c := installation.Default(declared)
	c.Spec.Infrastructure.Name = "signoz"

	return c
}

// clusteredCasting returns a bound Installation with the telemetry store and
// keeper cluster sizes overridden.
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

// The region is the one input that is neither declared nor discoverable, so it
// is all that is left in the tfvars.
func TestTfvarsTemplateCarriesTheRegion(t *testing.T) {
	casting := boundCasting(&installation.Casting{})
	casting.Metadata.Annotations = map[string]string{installation.ECSRegion.Key: "us-east-1"}

	buf := bytes.NewBuffer(nil)
	require.NoError(t, tfarsTF.Execute(buf, render(t, casting)))

	assert.JSONEq(t, `{"aws_region": "us-east-1"}`, buf.String())
}

// Nothing the substrate provisioned is named twice: the variables carry the
// names and tags the other casting derived from the same substrate name, and
// main.tf looks the objects up by them.
func TestSubstrateIsLookedUpThroughVariables(t *testing.T) {
	data := render(t, boundCasting(&installation.Casting{}))

	variables := bytes.NewBuffer(nil)
	require.NoError(t, variablesTF.Execute(variables, data))

	for _, expected := range []string{
		`"default": "signoz-cls"`,
		`"default": "signoz-sg-task"`,
		`"default": "signoz-installation-task"`,
		`"default": "signoz-installation-exec"`,
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
	casting.Metadata.Annotations = map[string]string{
		installation.ECSClusterARN.Key: "arn:aws:ecs:us-east-1:123456789012:cluster/test",
		installation.ECSSubnetIDs.Key:  "subnet-abc123, subnet-def456",
	}

	data := render(t, casting)

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
	assert.Contains(t, variables.String(), `"default": "signoz-installation-task"`)
}

func render(t *testing.T, casting *installation.Casting) templateData {
	t.Helper()

	data, err := New(slog.New(slog.DiscardHandler)).templateData(*casting)
	require.NoError(t, err)

	return data
}

func TestModulePlacement(t *testing.T) {
	t.Parallel()

	sqlite := boundCasting(&installation.Casting{})
	sqlite.Spec.MetaStore.Kind = installation.MetaStoreKindSQLite

	// The attribute clauses are the same match the substrate stamps on its
	// nodes, so a component reaches only the nodes of the substrate it is
	// bound to.
	seat := func(identity string) string {
		return `ec2InstanceId == '${local.seats[\"` + identity + `\"]}' and attribute:foundry.signoz.io/name == signoz and attribute:foundry.signoz.io/storage == persistent`
	}
	ephemeral := "attribute:foundry.signoz.io/name == signoz and attribute:foundry.signoz.io/storage == ephemeral"

	// Each stateful identity pins to the instance its claim resolved
	// (ec2InstanceId), with the storage attribute kept as a bootstrap
	// integrity check. Stateless services place onto the ephemeral pool.
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
			require.NoError(t, tc.template.Execute(buf, render(t, tc.casting)))
			out := buf.String()

			assert.Contains(t, out, `"launch_type": "EC2"`)
			for _, expression := range tc.expectedExpressions {
				assert.Contains(t, out, expression)
			}
			assert.NotContains(t, out, "capacity_provider")
		})
	}
}
