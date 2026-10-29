package dockerswarmcasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabels(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, installation.Default(&installation.Casting{}))
	require.NoError(t, err)

	compose, err := domain.NewYAMLMaterial(buf.Bytes(), "compose.yaml")
	require.NoError(t, err)

	for _, test := range []struct {
		name          string
		path          string
		pass          bool
		expectedValue string
	}{
		{"Kind_Valid", `services.ingester.labels.foundry\.signoz\.io/kind`, true, "Installation"},
		{"Name_Valid", `services.ingester.labels.foundry\.signoz\.io/name`, true, "signoz"},
		{"ManagedBy_Valid", `services.ingester.labels.foundry\.signoz\.io/managed-by`, true, "foundry"},
		{"Network_Invalid", `networks.signoz-network.labels.foundry\.signoz\.io/kind`, false, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, err := compose.GetBytes(test.path)

			if test.pass {
				require.NoError(t, err)
				assert.Equal(t, test.expectedValue, string(value))

				return
			}

			require.Error(t, err)
		})
	}
}
