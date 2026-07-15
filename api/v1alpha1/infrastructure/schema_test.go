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
				casting.Spec.Resource = v1alpha1.TypeResourceRef{
					APIVersion: "v1alpha1",
					Kind:       v1alpha1.KindInstallation,
					Name:       "signoz",
				}
			},
			pass: true,
		},
		{
			name: "ResourceAPIVersionOmitted_Valid",
			mutate: func(casting *Casting) {
				casting.Spec.Resource = v1alpha1.TypeResourceRef{
					Kind: v1alpha1.KindCollectionAgent,
					Name: "signoz-gateway",
				}
			},
			pass: true,
		},
		{
			name:   "ResourceMissing_Invalid",
			mutate: func(casting *Casting) {},
			pass:   false,
		},
		{
			name: "ResourceAPIVersionUnknown_Invalid",
			mutate: func(casting *Casting) {
				casting.Spec.Resource = v1alpha1.TypeResourceRef{
					APIVersion: "v2",
					Kind:       v1alpha1.KindInstallation,
					Name:       "signoz",
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
