package ecsterraformcasting

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const clusterARN = "arn:aws:ecs:us-east-1:123456789012:cluster/signoz"

// statedCasting states every annotation, so no role is created.
func statedCasting(t *testing.T) *collectionagent.Casting {
	t.Helper()

	config := collectionagent.Default()
	config.Metadata.Annotations = map[string]string{
		collectionagent.ECSRegion.Key:               "us-east-1",
		collectionagent.ECSClusterARN.Key:           clusterARN,
		collectionagent.ECSTaskRoleARN.Key:          "arn:aws:iam::123456789012:role/task",
		collectionagent.ECSTaskExecutionRoleARN.Key: "arn:aws:iam::123456789012:role/exec",
	}

	return config
}

// derivedCasting leaves the roles unstated, so the stack creates them.
func derivedCasting(t *testing.T) *collectionagent.Casting {
	t.Helper()

	config := collectionagent.Default()
	config.Metadata.Annotations = map[string]string{
		collectionagent.ECSRegion.Key:     "us-east-1",
		collectionagent.ECSClusterARN.Key: clusterARN,
	}

	return config
}

func templateDataFor(t *testing.T, config *collectionagent.Casting) templateData {
	t.Helper()

	return New(slog.New(slog.DiscardHandler)).templateData(*config)
}

func TestTemplatesRender(t *testing.T) {
	data := templateDataFor(t, statedCasting(t))

	for name, test := range map[string]struct {
		template *domain.Template
		data     any
	}{
		"Versions_Valid":  {versionsTF, data},
		"Providers_Valid": {providersTF, data},
		"Backend_Valid":   {backendTF, data},
		"Variables_Valid": {variablesTF, data},
		"Tfvars_Valid":    {tfvarsTF, data},
		"Main_Valid":      {mainTF, data},
		"Collector_Valid": {collectorTF, data},
		"Agent_Valid":     {agentYAMLTemplate, agentTemplateDataFor(*statedCasting(t))},
	} {
		t.Run(name, func(t *testing.T) {
			material, err := test.template.Render(test.data, strings.TrimSuffix(test.template.Name(), ".gotmpl"))

			require.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}

func TestTemplateData(t *testing.T) {
	for _, test := range []struct {
		name                string
		annotations         map[string]string
		expectedRegion      string
		expectedCluster     string
		expectedRolesStated bool
	}{
		{"AllStated_Valid", statedCasting(t).Metadata.Annotations, "us-east-1", clusterARN, true},
		{"RolesUnstated_Valid", derivedCasting(t).Metadata.Annotations, "us-east-1", clusterARN, false},
		{"NothingStated_Valid", nil, "", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Metadata.Annotations = test.annotations

			data := templateDataFor(t, config)

			assert.Equal(t, test.expectedRegion, data.Region)
			assert.Equal(t, test.expectedCluster, data.Cluster.Stated)

			assert.Equal(t, "signoz-collectionagent-iam-task", data.TaskRole.Name)
			assert.Equal(t, "signoz-collectionagent-iam-exec", data.ExecutionRole.Name)

			assert.Equal(t, test.expectedRolesStated, data.TaskRole.IsStated())
			assert.Equal(t, test.expectedRolesStated, data.ExecutionRole.IsStated())
		})
	}
}

// Terraform judges the platform inputs at plan, so an annotation-free casting
// still forges.
func TestUnstatedCastingForges(t *testing.T) {
	p := pourer.New("collectionagent")
	require.NoError(t, New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *collectionagent.Default(), p))

	materials, err := p.Pour()
	require.NoError(t, err)

	assert.NotEmpty(t, materials)
}

// A default in the variables would race the tfvars, so the root declares no
// variable the tfvars leaves unstated and none carries a default.
func TestVariablesAndTfvars(t *testing.T) {
	for _, test := range []struct {
		name         string
		config       *collectionagent.Casting
		expectedVars []string
	}{
		{"RolesUnstated_Valid", derivedCasting(t), []string{"aws_region", "cluster_arn", "task_role_name", "execution_role_name"}},
		{"RolesStated_Valid", statedCasting(t), []string{"aws_region", "cluster_arn", "task_role_arn", "execution_role_arn"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := templateDataFor(t, test.config)

			tfvars := bytes.NewBuffer(nil)
			require.NoError(t, tfvarsTF.Execute(tfvars, data))

			stated := map[string]string{}
			require.NoError(t, json.Unmarshal(tfvars.Bytes(), &stated))

			assert.ElementsMatch(t, test.expectedVars, slices.Collect(maps.Keys(stated)))

			for name, value := range stated {
				assert.NotEmpty(t, value, "%q is stated with no value", name)
			}

			variables := bytes.NewBuffer(nil)
			require.NoError(t, variablesTF.Execute(variables, data))

			var root struct {
				Variable map[string]map[string]any `json:"variable"`
			}
			require.NoError(t, json.Unmarshal(variables.Bytes(), &root))

			assert.ElementsMatch(t, test.expectedVars, slices.Collect(maps.Keys(root.Variable)))

			for name, variable := range root.Variable {
				assert.NotContains(t, variable, "default", "%q must carry no default", name)

				if strings.HasSuffix(name, "_name") {
					continue
				}

				assert.Contains(t, variable, "validation", "%q must carry a validation", name)
			}
		})
	}
}

func TestMainRoleOwnership(t *testing.T) {
	for _, test := range []struct {
		name                 string
		config               *collectionagent.Casting
		expectedRolesCreated bool
	}{
		{"RolesUnstated_Created", derivedCasting(t), true},
		{"RolesStated_Reused", statedCasting(t), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			require.NoError(t, mainTF.Execute(buf, templateDataFor(t, test.config)))

			material, err := domain.NewJSONMaterial(buf.Bytes(), "main.tf.json")
			require.NoError(t, err)

			_, err = material.GetBytes("resource.aws_iam_role")
			assert.Equal(t, test.expectedRolesCreated, err == nil)
		})
	}
}

func TestAgentConfig(t *testing.T) {
	material, err := agentYAMLTemplate.Render(agentTemplateDataFor(*statedCasting(t)), "agent.yaml")
	require.NoError(t, err)

	structured, ok := material.(domain.StructuredMaterial)
	require.True(t, ok)

	var config struct {
		Processors struct {
			Resourcedetection struct {
				Detectors []string `json:"detectors"`
			} `json:"resourcedetection"`
		} `json:"processors"`
		Receivers struct {
			Filelog struct {
				Operators []struct {
					Type    string `json:"type"`
					From    string `json:"from"`
					Field   string `json:"field"`
					If      string `json:"if"`
					Expr    string `json:"expr"`
					OnError string `json:"on_error"`
				} `json:"operators"`
			} `json:"filelog"`
		} `json:"receivers"`
	}
	require.NoError(t, json.Unmarshal(structured.JSONContents(), &config))

	// The ecs and docker detectors answer for the collector's own task, so on a
	// DAEMON they would stamp every container's telemetry with it.
	t.Run("Detectors_Valid", func(t *testing.T) {
		assert.Equal(t, []string{"env", "ec2"}, config.Processors.Resourcedetection.Detectors)
	})

	// A task without the labels option writes no attrs, and the miss must not be
	// logged once per operator per line.
	t.Run("MoveOperators_Valid", func(t *testing.T) {
		guarded := 0

		for _, operator := range config.Receivers.Filelog.Operators {
			if operator.Type != "move" {
				continue
			}

			if !strings.HasPrefix(operator.From, "body.attrs") && operator.From != "attributes.container_id" {
				continue
			}

			guarded++

			assert.Equal(t, "send_quiet", operator.OnError, "%q must tolerate a miss", operator.From)
		}

		assert.Equal(t, 5, guarded, "four ecs label lifts plus the carved container id")
	})

	// send_quiet still hands the error back to the file reader, so the operators
	// that read body.attrs must not run at all on a line that carries none.
	t.Run("AttrsOperatorsGuarded_Valid", func(t *testing.T) {
		guarded := 0

		for _, operator := range config.Receivers.Filelog.Operators {
			subject := operator.From
			if operator.Type == "remove" {
				subject = operator.Field
			}

			if operator.Type != "move" && operator.Type != "remove" {
				continue
			}

			if !strings.HasPrefix(subject, "body.attrs") {
				continue
			}

			guarded++

			assert.Equal(t, "body.attrs != nil", operator.If, "%q must be skipped without attrs", subject)
		}

		assert.Equal(t, 5, guarded, "four ecs label lifts plus the attrs removal")
	})

	// The agent's own containers log through json-file, so it would otherwise
	// ship its own lines back to itself.
	t.Run("OwnFamilyFiltered_Valid", func(t *testing.T) {
		expectedExpr := `body.attrs != nil && body.attrs["com.amazonaws.ecs.task-definition-family"] == "signoz-collector-agent"`

		filtered := 0

		for _, operator := range config.Receivers.Filelog.Operators {
			if operator.Type != "filter" {
				continue
			}

			filtered++

			assert.Equal(t, expectedExpr, operator.Expr)
			assert.Empty(t, operator.If, "stanza's filter never evaluates if")
			assert.Equal(t, "send", operator.OnError)
		}

		assert.Equal(t, 1, filtered, "the agent's own family is dropped once")
	})
}
