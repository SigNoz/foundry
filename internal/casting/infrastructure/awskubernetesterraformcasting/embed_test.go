package awskubernetesterraformcasting

import (
	"testing"

	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestTemplates_RenderValidJSON(t *testing.T) {
	data := &Data{
		Name:         "signoz",
		ResourceKind: "Installation",
		Persistent:   true,
		NodeGroups: []DataNodeGroup{
			{Name: "default", Storage: "persistent", MinSize: 2, MaxSize: 2, CPU: 2, Memory: 8, RootVolumeSize: 30, DataVolumeSize: 50},
		},
	}

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
			material, err := tc.template.Render(data, "out.tf.json")
			assert.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}
