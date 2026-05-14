package ecsterraformcasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNotEmptyAndValid(t *testing.T) {
	templates := map[string]*domain.Template{
		"mainTF":                  mainTF,
		"variablesTF":             variablesTF,
		"moduleMainTF":            moduleMainTF,
		"moduleVariablesTF":       moduleVariablesTF,
		"moduleOutputsTF":         moduleOutputsTF,
		"moduleTelemetryKeeperTF": moduleTelemetryKeeperTF,
		"moduleTelemetryStoreTF":  moduleTelemetryStoreTF,
		"moduleMigratorTF":        moduleMigratorTF,
		"moduleMetaStoreTF":       moduleMetaStoreTF,
		"moduleSignozTF":          moduleSignozTF,
		"moduleIngesterTF":        moduleIngesterTF,
	}

	for name, tmpl := range templates {
		assert.NotEmpty(t, tmpl, "%s should not be empty", name)
		buf := bytes.NewBuffer(nil)
		err := tmpl.Execute(buf, nil)
		assert.NoError(t, err, "error executing %s", name)
		assert.NotEmpty(t, buf.String(), "%s output should not be empty", name)
	}
}

func TestTfvarsTemplateWithAnnotations(t *testing.T) {
	assert.NotEmpty(t, tfarsTF)

	casting := &installation.Casting{
		CastingMeta: v1alpha1.CastingMeta{
			Metadata: v1alpha1.TypeMetadata{
				Name: "signoz",
				Annotations: map[string]string{
					"foundry.signoz.io/ecs/region":                  "us-east-1",
					"foundry.signoz.io/ecs/cluster-id":              "arn:aws:ecs:us-east-1:123456789012:cluster/test",
					"foundry.signoz.io/ecs/subnet-ids":              "subnet-abc123,subnet-def456",
					"foundry.signoz.io/ecs/security-group-ids":      "sg-abc123",
					"foundry.signoz.io/ecs/vpc-id":                  "vpc-abc123",
					"foundry.signoz.io/ecs/config-bucket":           "test-configs",
					"foundry.signoz.io/ecs/task-role-arn":           "arn:aws:iam::123456789012:role/task",
					"foundry.signoz.io/ecs/task-execution-role-arn": "arn:aws:iam::123456789012:role/exec",
					"foundry.signoz.io/ecs/capacity-provider":       "test-provider",
				},
			},
		},
	}

	buf := bytes.NewBuffer(nil)
	err := tfarsTF.Execute(buf, casting)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "us-east-1")
	assert.Contains(t, buf.String(), "subnet-abc123")
	assert.Contains(t, buf.String(), "subnet-def456")
}
