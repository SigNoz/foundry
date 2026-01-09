package dockercomposecasting

import (
	"io"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestNotEmpty(t *testing.T) {
	assert.NotEmpty(t, ComposeYAMLTemplate)
	config := v1alpha1.Casting{
		Metadata: v1alpha1.TypeMetadata{
			Name: "test",
		},
	}
	err := ComposeYAMLTemplate.Execute(io.Discard, config)
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	assert.NoError(t, err)
}
