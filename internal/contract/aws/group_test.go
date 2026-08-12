package aws

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceInstanceGroups(t *testing.T) {
	tests := []struct {
		name            string
		declaration     string
		pass            bool
		expectedSubnets map[string][]string
	}{
		{
			name: "NoStatedPlacement_EveryPrivateSubnet",
			declaration: `networking:
  subnets:
    private-a: {type: private, zone: us-east-1a}
    private-b: {type: private, zone: us-east-1b}
    public-a: {type: public, zone: us-east-1a}
instanceGroups:
  ephemeral: {storage: ephemeral, minSize: 1, maxSize: 1}
`,
			pass:            true,
			expectedSubnets: map[string][]string{"ephemeral": {"private-a", "private-b"}},
		},
		{
			name: "StatedPlacement_Honoured",
			declaration: `networking:
  subnets:
    private-a: {type: private, zone: us-east-1a}
    private-b: {type: private, zone: us-east-1b}
instanceGroups:
  persistent: {storage: persistent, minSize: 3, maxSize: 3, subnets: [private-b]}
`,
			pass:            true,
			expectedSubnets: map[string][]string{"persistent": {"private-b"}},
		},
		{
			name: "NoPrivateSubnet_Invalid",
			declaration: `networking:
  subnets:
    public-a: {type: public, zone: us-east-1a}
instanceGroups:
  ephemeral: {storage: ephemeral, minSize: 1, maxSize: 1}
`,
			pass: false,
		},
		{
			name: "MalformedGroupKey_Invalid",
			declaration: `networking:
  subnets:
    private-a: {type: private, zone: us-east-1a}
instanceGroups:
  Ephemeral: {storage: ephemeral, minSize: 1, maxSize: 1}
`,
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			declaration := &infrastructure.ResourceConfig{}
			require.NoError(t, domain.UnmarshalYAML([]byte(tt.declaration), declaration))

			placed, err := PlaceInstanceGroups(declaration)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			for _, placement := range placed {
				assert.Equal(t, tt.expectedSubnets[placement.Key], placement.Subnets)
				assert.Equal(t, placement.Storage, placement.Group.Storage())
				assert.Equal(t, placement.Declared.Storage, placement.Storage.String())
				assert.Equal(t, placement.Key, placement.Group.Key().String())
			}
		})
	}
}
