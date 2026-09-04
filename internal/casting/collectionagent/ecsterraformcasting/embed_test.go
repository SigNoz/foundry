package ecsterraformcasting

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statedCasting is a casting whose every axis is stated, so no role is created.
func statedCasting(t *testing.T) *collectionagent.Casting {
	t.Helper()

	config := collectionagent.Default()
	config.Metadata.Annotations = map[string]string{
		collectionagent.ECSRegion.Key:               "us-east-1",
		collectionagent.ECSClusterARN.Key:           "arn:aws:ecs:us-east-1:123456789012:cluster/signoz",
		collectionagent.ECSTaskRoleARN.Key:          "arn:aws:iam::123456789012:role/task",
		collectionagent.ECSTaskExecutionRoleARN.Key: "arn:aws:iam::123456789012:role/exec",
	}

	return config
}

// derivedCasting states only what has nothing to derive it, so roles are created.
func derivedCasting(t *testing.T) *collectionagent.Casting {
	t.Helper()

	config := collectionagent.Default()
	config.Metadata.Annotations = map[string]string{
		collectionagent.ECSRegion.Key:     "us-east-1",
		collectionagent.ECSClusterARN.Key: "arn:aws:ecs:us-east-1:123456789012:cluster/signoz",
	}

	return config
}

func data(t *testing.T, config *collectionagent.Casting) templateData {
	t.Helper()

	data, err := New(nil).templateData(*config)
	require.NoError(t, err)

	return data
}

func TestNotEmptyAndValid(t *testing.T) {
	for _, config := range map[string]*collectionagent.Casting{
		"Stated":  statedCasting(t),
		"Derived": derivedCasting(t),
	} {
		for name, tmpl := range map[string]*domain.Template{
			"versionsTF":  versionsTF,
			"providersTF": providersTF,
			"backendTF":   backendTF,
			"variablesTF": variablesTF,
			"tfvarsTF":    tfvarsTF,
			"mainTF":      mainTF,
			"collectorTF": collectorTF,
		} {
			buf := bytes.NewBuffer(nil)
			err := tmpl.Execute(buf, data(t, config))

			assert.NoError(t, err, "error executing %s", name)
			assert.NotEmpty(t, buf.String(), "%s output should not be empty", name)
		}
	}

	assert.NotEmpty(t, agentYAMLTemplate)

	buf := bytes.NewBuffer(nil)
	err := agentYAMLTemplate.Execute(buf, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestTemplateData(t *testing.T) {
	for _, test := range []struct {
		name        string
		annotations map[string]string
		pass        bool
	}{
		{
			name:        "AllStated_Valid",
			annotations: statedCasting(t).Metadata.Annotations,
			pass:        true,
		},
		{
			name:        "RolesUnstated_Valid",
			annotations: derivedCasting(t).Metadata.Annotations,
			pass:        true,
		},
		{
			name:        "UnstatedRegion_Invalid",
			annotations: map[string]string{collectionagent.ECSClusterARN.Key: "arn:aws:ecs:us-east-1:123456789012:cluster/signoz"},
			pass:        false,
		},
		{
			name:        "UnstatedCluster_Invalid",
			annotations: map[string]string{collectionagent.ECSRegion.Key: "us-east-1"},
			pass:        false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Metadata.Annotations = test.annotations

			_, err := New(nil).templateData(*config)

			if !test.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestCollectorTemplateAppConfigDelivery(t *testing.T) {
	config := statedCasting(t)
	config.Spec.Collector.Spec.Config.Data = map[string]string{
		config.Spec.Collector.Kind.ConfigKey(): "receivers: {}\n",
	}

	buf := bytes.NewBuffer(nil)
	require.NoError(t, collectorTF.Execute(buf, data(t, config)))

	material, err := domain.NewJSONMaterial(buf.Bytes(), "collector.tf.json")
	require.NoError(t, err)

	for path, expected := range map[string]string{
		"resource.aws_ecs_service.collector.scheduling_strategy": "DAEMON",

		// A daemon rolls a revision only if it may drop to zero healthy on an
		// instance, and a bad revision has to roll itself back.
		"resource.aws_ecs_service.collector.deployment_minimum_healthy_percent":  "0",
		"resource.aws_ecs_service.collector.deployment_circuit_breaker.enable":   "true",
		"resource.aws_ecs_service.collector.deployment_circuit_breaker.rollback": "true",

		"resource.aws_ecs_task_definition.collector.network_mode":                    "host",
		"resource.aws_appconfig_configuration_profile.collector.name":                "collector-agent",
		"resource.aws_appconfig_configuration_profile.collector.location_uri":        "hosted",
		"resource.aws_appconfig_hosted_configuration_version.collector.content_type": "application/x-yaml",

		// json-file on either container makes the collector tail its own log.
		"locals.containers_collector.0.logConfiguration.logDriver": "none",
		"locals.containers_collector.1.logConfiguration.logDriver": "none",
	} {
		value, err := material.GetBytes(path)

		assert.NoError(t, err, "reading %s", path)
		assert.Equal(t, expected, string(value), "at %s", path)
	}

	// The provider refuses deployment_maximum_percent on a DAEMON service.
	_, err = material.GetBytes("resource.aws_ecs_service.collector.deployment_maximum_percent")
	assert.Error(t, err, "deployment_maximum_percent is not valid with DAEMON")

	// The config reaches the task through the AppConfig agent, never a bucket.
	assert.Contains(t, buf.String(), "aws-appconfig-agent")
	assert.Contains(t, buf.String(), "FOUNDRY_CONFIG_DIGEST")
	assert.NotContains(t, buf.String(), "aws_s3_object")

	// A changed config has to replace the task, so the digest is in the
	// definition.
	other := statedCasting(t)
	other.Spec.Collector.Spec.Config.Data = map[string]string{
		other.Spec.Collector.Kind.ConfigKey(): "receivers: {otlp: {}}\n",
	}

	otherBuf := bytes.NewBuffer(nil)
	require.NoError(t, collectorTF.Execute(otherBuf, data(t, other)))

	assert.NotEqual(t, buf.String(), otherBuf.String())
}

func TestMainTemplateRoleOwnership(t *testing.T) {
	derived := bytes.NewBuffer(nil)
	require.NoError(t, mainTF.Execute(derived, data(t, derivedCasting(t))))

	assert.Contains(t, derived.String(), "aws_iam_role")
	assert.Contains(t, derived.String(), "appconfig:StartConfigurationSession")
	assert.Contains(t, derived.String(), v1alpha1.LabelManagedBy.Value)

	stated := bytes.NewBuffer(nil)
	require.NoError(t, mainTF.Execute(stated, data(t, statedCasting(t))))

	// Nothing is created for a role the operator brought.
	assert.NotContains(t, stated.String(), "aws_iam_role")
	assert.NotContains(t, stated.String(), "appconfig:StartConfigurationSession")
	assert.Contains(t, stated.String(), "aws_appconfig_application")

	material, err := domain.NewJSONMaterial(stated.Bytes(), "main.tf.json")
	require.NoError(t, err)

	for path, expected := range map[string]string{
		"resource.aws_appconfig_application.main.name":         "signoz-collectionagent-appconfig",
		"resource.aws_appconfig_deployment_strategy.main.name": "signoz-collectionagent-appconfig-strategy",
	} {
		value, err := material.GetBytes(path)

		assert.NoError(t, err, "reading %s", path)
		assert.Equal(t, expected, string(value), "at %s", path)
	}
}

// The ecs and docker detectors answer for the collector's own task, so on a
// DAEMON they stamp every container's telemetry with it.
func TestAgentConfigDetectors(t *testing.T) {
	material, err := agentYAMLTemplate.Render(nil, "agent.yaml")
	require.NoError(t, err)

	structured, ok := material.(domain.StructuredMaterial)
	require.True(t, ok)

	for path, expected := range map[string]string{
		"processors.resourcedetection.detectors.0": "env",
		"processors.resourcedetection.detectors.1": "ec2",
	} {
		value, err := structured.GetBytes(path)

		assert.NoError(t, err, "reading %s", path)
		assert.Equal(t, expected, string(value), "at %s", path)
	}

	_, err = structured.GetBytes("processors.resourcedetection.detectors.2")
	assert.Error(t, err, "detectors must hold exactly env and ec2")
}

// A task without the labels option writes no attrs. Those records must still
// arrive with container.id, and the miss must not be logged.
func TestAgentConfigMoveOperatorsTolerateMissingLabels(t *testing.T) {
	material, err := agentYAMLTemplate.Render(nil, "agent.yaml")
	require.NoError(t, err)

	structured, ok := material.(domain.StructuredMaterial)
	require.True(t, ok)

	guarded := 0

	for i := range 32 {
		kind, err := structured.GetBytes(fmt.Sprintf("receivers.filelog.operators.%d.type", i))
		if err != nil {
			break
		}

		if string(kind) != "move" {
			continue
		}

		from, err := structured.GetBytes(fmt.Sprintf("receivers.filelog.operators.%d.from", i))
		require.NoError(t, err, "operator %d is a move with no from", i)

		// body.stream, body.log and log.file.path are always present on a
		// docker json line; only the label lifts and the carved id can miss.
		source := string(from)
		if !strings.HasPrefix(source, "body.attrs") && source != "attributes.container_id" {
			continue
		}

		guarded++

		onError, err := structured.GetBytes(fmt.Sprintf("receivers.filelog.operators.%d.on_error", i))
		if !assert.NoError(t, err, "%q must tolerate a miss", source) {
			continue
		}

		// send would log one error per operator per line for every unlabelled task.
		assert.Equal(t, "send_quiet", string(onError), "at %q", source)
	}

	assert.Equal(t, 5, guarded, "four ecs label lifts plus the carved container id")
}
