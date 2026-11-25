package kuberneteshelmcasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNotEmptyAndValid(t *testing.T) {
	templates := map[string]*domain.Template{
		"valuesYAMLTemplate":    valuesYAMLTemplate,
		"mcpValuesYAMLTemplate": mcpValuesYAMLTemplate,
	}

	data := *installation.Default(&installation.Casting{})

	for name, tmpl := range templates {
		assert.NotEmpty(t, tmpl, "%s should not be empty", name)
		buf := bytes.NewBuffer(nil)
		err := tmpl.Execute(buf, data)
		assert.NoError(t, err, "error executing %s", name)
		assert.NotEmpty(t, buf.String(), "%s output should not be empty", name)
	}
}
