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
			name: "ResourceProvided_Valid",
			mutate: func(casting *Casting) {
				casting.Spec.Resource.Kind = ResourceKindInstallation
				casting.Spec.Resource.Spec.Name = "signoz"
			},
			pass: true,
		},
		{
			name: "CollectionAgentResource_Valid",
			mutate: func(casting *Casting) {
				casting.Spec.Resource.Kind = ResourceKindCollectionAgent
				casting.Spec.Resource.Spec.Name = "signoz-gateway"
			},
			pass: true,
		},
		{
			name:   "ResourceMissing_Invalid",
			mutate: func(casting *Casting) {},
			pass:   false,
		},
		{
			name: "ResourceKindMissing_Invalid",
			mutate: func(casting *Casting) {
				casting.Spec.Resource.Spec.Name = "signoz"
			},
			pass: false,
		},
		{
			name: "ResourceNameMissing_Invalid",
			mutate: func(casting *Casting) {
				casting.Spec.Resource.Kind = ResourceKindInstallation
			},
			pass: false,
		},
		{
			name: "ResourceAPIVersionMissing_Invalid",
			mutate: func(casting *Casting) {
				casting.Spec.Resource = Resource{
					Kind: ResourceKindInstallation,
					Spec: ResourceSpec{
						TypeCastingRefSpec: v1alpha1.TypeCastingRefSpec{Name: "signoz"},
					},
				}
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
