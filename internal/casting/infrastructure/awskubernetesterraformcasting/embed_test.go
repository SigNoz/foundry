package awskubernetesterraformcasting

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestTemplates_RenderValidJSON(t *testing.T) {
	config := infrastructure.Default()
	config.Spec.Resource.Kind = infrastructure.ResourceKindInstallation
	config.Spec.Resource.Spec.Name = "signoz"

	testCases := []struct {
		name     string
		template *domain.Template
	}{
		{name: "ProvidersTemplate_RendersValidJSON", template: providersTFTemplate},
		{name: "MainTemplate_RendersValidJSON", template: mainTFTemplate},
		{name: "VariablesTemplate_RendersValidJSON", template: variablesTFTemplate},
		{name: "OutputsTemplate_RendersValidJSON", template: outputsTFTemplate},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			material, err := tc.template.Render(*config, "out.tf.json")
			assert.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}
