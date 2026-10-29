package kuberneteskustomizecasting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithoutNamespaces(t *testing.T) {
	namespaceYAML := "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: signoz\n"
	deploymentYAML := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: signoz-ingester\n  namespace: signoz\n"
	serviceYAML := "apiVersion: v1\nkind: Service\nmetadata:\n  name: signoz-ingester\n  namespace: signoz\n"

	tests := []struct {
		name              string
		manifests         string
		expectedManifests string
		pass              bool
	}{
		{name: "Namespace_Dropped", manifests: namespaceYAML + "---\n" + deploymentYAML, expectedManifests: deploymentYAML, pass: true},
		{name: "None_Unchanged", manifests: deploymentYAML + "---\n" + serviceYAML, expectedManifests: deploymentYAML + "---\n" + serviceYAML, pass: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifests, err := withoutNamespaces([]byte(tt.manifests))
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedManifests, string(manifests))
		})
	}
}
