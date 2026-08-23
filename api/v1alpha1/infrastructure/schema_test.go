package infrastructure

import (
	"encoding/json"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"

	"github.com/stretchr/testify/assert"
)

func TestSchema(t *testing.T) {
	assert.NotNil(t, Schema())
}

func TestSchemaValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(casting *Casting)
		pass   bool
	}{
		{
			name: "Deployment_Valid",
			mutate: func(casting *Casting) {
				casting.Spec.Deployment = v1alpha1.TypeDeployment{
					Platform: v1alpha1.PlatformECS,
					Mode:     v1alpha1.ModeEC2,
					Flavor:   v1alpha1.FlavorTerraform,
				}
			},
			pass: true,
		},
		{
			name: "NameMissing_Invalid",
			mutate: func(casting *Casting) {
				casting.Spec.Deployment = v1alpha1.TypeDeployment{
					Platform: v1alpha1.PlatformECS,
					Mode:     v1alpha1.ModeEC2,
					Flavor:   v1alpha1.FlavorTerraform,
				}
				casting.Metadata.Name = ""
			},
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			casting := Default()
			tt.mutate(casting)

			contents, err := json.Marshal(casting)
			assert.NoError(t, err)

			payload := map[string]any{}
			assert.NoError(t, json.Unmarshal(contents, &payload))

			err = Schema().Validate(payload)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
